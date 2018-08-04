package bot

import (
	"fmt"
	"log"
	"os"
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
	blindTestChannelID string
	logger             *log.Logger
	users              map[string]*user
	entries            map[string]*entry
	entriesByThreadID  map[string]*entry

	botUserClient, client *slack.Client
	rtm                   *slack.RTM
	db                    *db.DB
}

type SlackMessage struct {
	Msg    string
	TeamID int
}

func New(debug bool, botUserKey, key, masterEmail, botName, BlindTestChannel, dbPath string, domain []string) (*BlindBot, error) {
	var err error
	db := InitDB(dbPath)
	b := &BlindBot{
		users:         make(map[string]*user),
		name:          botName,
		domain:        domain[0],
		botUserClient: slack.New(botUserKey),
		client:        slack.New(key),
		logger:        log.New(os.Stdout, "slack-bot-"+botName+": ", log.Lshortfile|log.LstdFlags),
		db:            db,
	}

	// load entries in memory
	b.scanEntriesFromdb()

	// set up Slack logger
	slack.SetLogger(b.logger)
	b.botUserClient.SetDebug(debug)

	// load users in memory from Slack API
	// TODO: load users from database
	err = b.scanUsers(masterEmail, botName)
	if err != nil {
		return nil, err
	}

	// find blindtest channel ID
	// TODO: use another method for that
	channels, err := b.botUserClient.GetChannels(true)
	if err != nil {
		return nil, err
	}
	for _, channel := range channels {
		if channel.Name == BlindTestChannel {
			b.blindTestChannelID = channel.ID
			return b, nil
		}
	}

	return nil, ErrBlindTestChannelNotFound
}

func (b *BlindBot) Run() {
	b.rtm = b.botUserClient.NewRTM()
	go b.rtm.ManageConnection()
	for msg := range b.rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:

			// this part will be deprecated when users are stored in the database
			for userID := range b.users {
				_, _, channel, err := b.rtm.OpenIMChannel(userID)
				if err != nil {
					log.Println("cannot get channel ID for user "+b.getUsername(userID), err)
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

		case *slack.TeamJoinEvent:
			// TODO: add user to db / in memory

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

func (b *BlindBot) announce(v interface{}, channelIDs ...string) ([]string, error) {
	threadIDs := []string{}
	s := fmt.Sprintf("%v", v)
	for _, channelID := range channelIDs {
		params := slack.NewPostMessageParameters()
		params.AsUser = true
		params.LinkNames = 1
		params.UnfurlLinks = true
		_, threadID, err := b.botUserClient.PostMessage(channelID, s, params)
		if err != nil {
			return nil, err
		}
		threadIDs = append(threadIDs, threadID)
	}
	return threadIDs, nil
}

func (b *BlindBot) masterChannelID() string {
	return b.users[b.masterID].channelID
}
