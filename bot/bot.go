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
	domain             string
	blindTestChannelID string
	logger             *log.Logger
	users              map[string]*user
	entries            map[string]*entry

	client *slack.Client
	rtm    *slack.RTM
	db     *db.DB
}

type SlackMessage struct {
	Msg    string
	TeamID int
}

func New(debug bool, key, masterEmail, domain, botName, BlindTestChannel string, db *db.DB) (*BlindBot, error) {
	var err error
	b := &BlindBot{
		users:   make(map[string]*user),
		entries: scanEntries(db),
		domain:  domain,
		client:  slack.New(key),
		logger:  log.New(os.Stdout, "slack-bot-"+botName+": ", log.Lshortfile|log.LstdFlags),
		db:      db,
	}

	slack.SetLogger(b.logger)
	b.client.SetDebug(debug)

	// scan existing users
	err = b.scanUsers(masterEmail, botName)
	if err != nil {
		return nil, err
	}

	channels, err := b.client.GetChannels(true)
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
	b.rtm = b.client.NewRTM()
	go b.rtm.ManageConnection()
	for msg := range b.rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			for userID := range b.users {
				_, _, channel, err := b.rtm.OpenIMChannel(userID)
				if err != nil {
					log.Println("cannot get channel ID for user " + b.getUsername(userID))
					continue
				}
				b.users[userID].channelID = channel
			}
			b.log("Hello Master.")

		case *slack.MessageEvent:
			if ev.SubType == "" && strings.Contains(ev.Text, "<@"+b.id+">") {
				go b.submitWithLogs(ev.Text, ev.User)
			}
			if ev.Channel == b.masterChannelID() && strings.Contains(ev.Text, "show entries") {
				b.Lock()
				for _, entry := range b.entries {
					b.announce(entry, b.masterChannelID())
				}
				b.Unlock()
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

func (b *BlindBot) announce(v interface{}, channelIDs ...string) {
	s := fmt.Sprintf("%v", v)
	for _, channelID := range channelIDs {
		b.rtm.SendMessage(b.rtm.NewOutgoingMessage(s, channelID))
	}
}

func (b *BlindBot) masterChannelID() string {
	return b.users[b.masterID].channelID
}
