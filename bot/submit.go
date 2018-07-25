package bot

import (
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

var (
	youtubeURL, _ = regexp.Compile(`.<*(https?\:\/\/)?(www\.)?(youtube\.com|youtu\.?be)\/(.+)>.*`)
	videoID, _    = regexp.Compile(`^(watch\?v=)?(.*)$`)
)

func (b *Bot) youtubeURL(ev *slack.MessageEvent) {
	if ev.SubType != "" || !strings.Contains(ev.Text, "<@"+b.me.ID+">") {
		return
	}
	matches := youtubeURL.FindStringSubmatch(ev.Text)
	if len(matches) != 0 {
		user := b.users[ev.User]
		user.Lock()
		user.requestLimit++
		youtubeID := videoID.FindStringSubmatch(matches[4])[2]
		if user.requestLimit < 10 {
			_, _, channel, err := b.RTM.OpenIMChannel(ev.User)
			if err != nil {
				log.Println(err)
			} else {
				b.RTM.SendMessage(b.RTM.NewOutgoingMessage(youtubeID, channel))
			}
			//_, err := b.RTM.UploadFile(slack.FileUploadParameters{}})
			//if err != nil {
			//	log.Println(err)
			//}
		} else {
			_, _, channel, err := b.RTM.OpenIMChannel(ev.User)
			if err != nil {
				log.Println(err)
			} else {
				b.RTM.SendMessage(b.RTM.NewOutgoingMessage(strconv.Itoa(user.requestLimit)+" requests in a minute, slow down!", channel))
			}
		}
		go user.decreaseLimitTimeout()
		user.Unlock()
	}
}

func (u *user) decreaseLimitTimeout() {
	time.Sleep(time.Minute)
	u.Lock()
	u.requestLimit--
	u.Unlock()
}
