package bot

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/nlopes/slack"
)

var (
	ErrBlindTestChannelNotFound = fmt.Errorf("BlindTest channel not found")
)

type BlindBot struct {
	sync.Mutex
	masterID           string
	id                 string
	name               string
	domain             string
	domainRegex        *regexp.Regexp
	blindTestChannelID string
	logger             *log.Logger
	users              map[string]*user
	entries            map[string]*entry
	entriesByThreadID  map[string]*entry

	writeClient, readClient *slack.Client
	rtm                     *slack.RTM
	db                      *db.DB
}

type SlackMessage struct {
	Msg    string
	TeamID int
}

func New(debug bool, key, oauth2key, masterEmail, botName, BlindTestChannel, dbPath string, domain []string) (*BlindBot, error) {
	var err error
	db := InitDB(dbPath)
	b := &BlindBot{
		users:             make(map[string]*user),
		entriesByThreadID: make(map[string]*entry),
		entries:           scanEntriesFromdb(db.Use(EntryCollection)),
		name:              botName,
		domain:            domain[0],
		domainRegex:       regexp.MustCompile(strings.Replace(domain[0], ".", `\.`, -1) + `\/music\/(.*)$`),
		writeClient:       slack.New(key),
		readClient:        slack.New(oauth2key),
		logger:            log.New(os.Stdout, "slack-bot-"+botName+": ", log.Lshortfile|log.LstdFlags),
		db:                db,
	}

	slack.SetLogger(b.logger)
	b.writeClient.SetDebug(debug)

	// TODO: store users in database
	// scan existing users
	err = b.scanUsers(masterEmail, botName)
	if err != nil {
		return nil, err
	}

	channels, err := b.writeClient.GetChannels(true)
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		if channel.Name == BlindTestChannel {
			b.blindTestChannelID = channel.ID
			b.syncEntries()
			return b, nil
		}
	}

	return nil, ErrBlindTestChannelNotFound
}

func (b *BlindBot) Run() {
	b.rtm = b.writeClient.NewRTM()
	go b.rtm.ManageConnection()
	for msg := range b.rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:

			// this part will be deprecated when users are stored in the database
			for userID := range b.users {
				_, _, channel, err := b.rtm.OpenIMChannel(userID)
				if err != nil {
					log.Println("cannot get channel ID for user " + b.getUsername(userID))
					continue
				}
				b.users[userID].channelID = channel
			}
			// end of deprecation

			b.log("Hello Master.")

		case *slack.MessageEvent:
			if ev.SubType == "" && strings.Contains(ev.Text, "<@"+b.id+">") {
				go b.submitWithLogs(ev.Text, ev.User)
			}
			if ev.Channel == b.blindTestChannelID && ev.ThreadTimestamp != ev.Timestamp && ev.ThreadTimestamp != "" {
				go b.log(b.validateAnswer(ev))
			}

			if ev.Channel == b.masterChannelID() {
				go b.log(b.masterCommands(ev))
			}
		case *slack.InvalidAuthEvent:
			b.logger.Printf("Invalid credentials")
			return
		default:
		}
	}
}

func (b *BlindBot) log(v interface{}, userIDs ...string) {
	if v == nil {
		return
	}
	s := fmt.Sprintf("%v", v)

	// log to Users
	usersNicknames := ""
	for _, userID := range userIDs {
		b.rtm.SendMessage(b.rtm.NewOutgoingMessage(s, b.users[userID].channelID))
		usersNicknames += b.getUsername(userID) + ", "
	}

	// log to console
	s = usersNicknames + s
	log.Println(s)

	// log to Master
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(s, b.masterChannelID()))
}

func (b *BlindBot) announce(v interface{}, channelIDs ...string) []string {
	threadIDs := []string{}
	s := fmt.Sprintf("%v", v)
	for _, channelID := range channelIDs {
		params := slack.NewPostMessageParameters()
		params.AsUser = true
		params.LinkNames = 1
		_, threadID, _ := b.writeClient.PostMessage(channelID, s, params)
		threadIDs = append(threadIDs, threadID)
	}
	return threadIDs
}

func (b *BlindBot) masterChannelID() string {
	return b.users[b.masterID].channelID
}
