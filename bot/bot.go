package bot

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/nlopes/slack"
)

var (
	ErrBlindTestChannelNotFound = fmt.Errorf("BlindTest channel not found")
)

type Bot struct {
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
}

type SlackMessage struct {
	Msg    string
	TeamID int
}

func New(debug bool, key, masterEmail, domain, botName, BlindTestChannel string) (*Bot, error) {
	var err error
	bot := &Bot{
		users:   make(map[string]*user),
		entries: scanEntries(),
		domain:  domain,
		client:  slack.New(key),
		logger:  log.New(os.Stdout, "slack-bot-"+botName+": ", log.Lshortfile|log.LstdFlags),
	}

	slack.SetLogger(bot.logger)
	bot.client.SetDebug(debug)

	// scan existing users
	err = bot.scanUsers(masterEmail, botName)
	if err != nil {
		return nil, err
	}

	channels, err := bot.client.GetChannels(true)
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		if channel.Name == BlindTestChannel {
			bot.blindTestChannelID = channel.ID
			return bot, nil
		}
	}

	return nil, ErrBlindTestChannelNotFound
}

func (b *Bot) Run() {
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
		case *slack.InvalidAuthEvent:
			b.logger.Printf("Invalid credentials")
			return
		default:
		}
	}
}

func (b *Bot) log(v interface{}, userIDs ...string) {
	s := fmt.Sprintf("%v", v)

	// log to Users
	usersNicknames := ""
	for _, userID := range userIDs {
		b.rtm.SendMessage(b.rtm.NewOutgoingMessage(s, b.users[userID].channelID))
		usersNicknames += b.getUsername(userID) + ", "
	}

	// log to console
	s = usersNicknames + s
	log.Println(s, v)

	// log to Master
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(s, b.masterChannelID()))
}

func (b *Bot) announce(v interface{}, channelIDs ...string) {
	s := fmt.Sprintf("%v", v)
	for _, channelID := range channelIDs {
		b.rtm.SendMessage(b.rtm.NewOutgoingMessage(s, channelID))
	}
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(s, b.masterChannelID()))
}

func (b *Bot) masterChannelID() string {
	return b.users[b.masterID].channelID
}
