package bot

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/nlopes/slack"
)

type Bot struct {
	MasterChannelID string
	BTChannel       slack.Channel
	Logger          *log.Logger

	client   *slack.Client
	RTM      *slack.RTM
	master   slack.User
	teamInfo *slack.TeamInfo
	users    map[string]*user
	me       slack.User
}

type SlackMessage struct {
	Msg    string
	TeamID int
}

type user struct {
	sync.Mutex
	name               string
	requestVeilleCount int
	requestLimit       int
}

func New(debug bool, key, master, botName, BTChannel string) (*Bot, error) {
	var err error
	bot := &Bot{users: make(map[string]*user)}
	bot.client = slack.New(key)
	bot.Logger = log.New(os.Stdout, "slack-bot-"+botName+": ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(bot.Logger)
	bot.client.SetDebug(debug)
	users, err := bot.client.GetUsers()
	if err != nil {
		return nil, err
	}
	errMasterNotFound := fmt.Errorf("Master not found")
	for _, u := range users {
		if u.Profile.Email == master {
			bot.master = u
			errMasterNotFound = nil
		}
		if u.IsBot && u.Name == botName {
			bot.me = u
		}
		bot.users[u.ID] = &user{
			name: u.Name,
		}
	}
	if errMasterNotFound != nil {
		return nil, errMasterNotFound
	}

	channels, err := bot.client.GetChannels(true)
	if err != nil {
		return nil, err
	}
	errVeilleChan := fmt.Errorf(BTChannel + " channel not found")
	for _, channel := range channels {
		if channel.Name == BTChannel {
			bot.BTChannel = channel
			errVeilleChan = nil
		}
	}
	if errVeilleChan != nil {
		return nil, errVeilleChan
	}

	bot.teamInfo, err = bot.client.GetTeamInfo()
	if err != nil {
		return nil, err
	}

	return bot, nil
}

func (b *Bot) Run() {
	b.RTM = b.client.NewRTM()
	go b.RTM.ManageConnection()
	for msg := range b.RTM.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello

		case *slack.ConnectedEvent:
			log.Println("Connected to " + b.teamInfo.Name)
			_, _, channel, err := b.RTM.OpenIMChannel(b.master.ID)
			if err != nil {
				panic(err)
			}
			b.MasterChannelID = channel
			b.RTM.SendMessage(b.RTM.NewOutgoingMessage("Hello master", b.MasterChannelID))

		case *slack.MessageEvent:
			b.youtubeURL(ev)

		case *slack.PresenceChangeEvent:

		case *slack.LatencyReport:

		case *slack.RTMError:

		case *slack.InvalidAuthEvent:
			b.Logger.Printf("Invalid credentials")
			return

		default:

		}
	}
}
