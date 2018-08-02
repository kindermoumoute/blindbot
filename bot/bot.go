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

func New(debug bool, key, oauth2key, masterEmail, domain, botName, BlindTestChannel string, db *db.DB) (*BlindBot, error) {
	var err error
	b := &BlindBot{
		users:             make(map[string]*user),
		entriesByThreadID: make(map[string]*entry),
		entries:           scanEntries(db),
		domain:            domain,
		domainRegex:       regexp.MustCompile(strings.Replace(domain, ".", `\.`, -1) + `\/music\/(.*)$`),
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

			// this part will be deprecated when the database is updated
			for i := 0; i < 4; i++ {
				searchParams := slack.NewSearchParameters()
				searchParams.Page = i + 1
				history, err := b.readClient.SearchMessages("from:"+botName+" in:"+BlindTestChannel, searchParams)
				if err != nil {
					log.Println(err)
				}

				for _, message := range history.Matches {
					log.Printf("Message \"%s\" %s \n", message.Text, message.User)
				}
				for _, entry := range b.entries {
					if entry.threadID == "" {
						log.Printf("Entry %s has no threadID\n", entry.hashedYoutubeID)
						for _, message := range history.Matches {
							if message.User == b.id && strings.Contains(message.Text, entry.Path()) {
								log.Println("Updating threadID for ", entry.hashedYoutubeID)
								b.updateThread(entry, message.Timestamp)
							}
						}
					}
				}
			}
			// end of deprecation
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
			if ev.BotID == b.id && ev.Channel == b.blindTestChannelID {
				go b.log(b.newChallenge(ev))
			}
			if ev.Channel == b.blindTestChannelID && ev.ThreadTimestamp != ev.Timestamp && ev.ThreadTimestamp != "" {
				go b.log(b.validateAnswer(ev))
			}

			if ev.Channel == b.masterChannelID() && strings.Contains(ev.Text, "show entries") {
				go func() {
					b.Lock()
					message := ""
					count := 0
					for _, entry := range b.entries {
						message += entry.String() + "\n"
						count++
						if count%50 == 0 {
							b.announce(message, b.masterChannelID())
							message = ""
						}
					}

					b.announce(message, b.masterChannelID())
					b.Unlock()
				}()
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
