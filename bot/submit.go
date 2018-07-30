package bot

import (
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/rylio/ytdl"
)

var (
	youtubeURL, _ = regexp.Compile(`.<*(https?\:\/\/)?(www\.)?(youtube\.com|youtu\.?be)\/(.+)>.*`)
	videoID, _    = regexp.Compile(`^(watch\?v=)?(.*)$`)
)

func (b *Bot) logger(channel string) func(string, error) {
	return func(s string, err error) {
		b.RTM.SendMessage(b.RTM.NewOutgoingMessage(s, channel))
		if err != nil {
			s += ", error: " + err.Error()
		}
		log.Println(s, err)
		b.RTM.SendMessage(b.RTM.NewOutgoingMessage(s, b.MasterChannelID))
	}
}

func (b *Bot) youtubeURL(ev *slack.MessageEvent) {
	if ev.SubType != "" || !strings.Contains(ev.Text, "<@"+b.me.ID+">") {
		return
	}
	logInSlack := b.logger(ev.Channel)
	matches := youtubeURL.FindStringSubmatch(ev.Text)
	if len(matches) != 0 {
		user := b.users[ev.User]
		user.Lock()
		user.requestLimit++
		defer func() {
			time.Sleep(time.Second) // avoid the user to submit twice in a second
			user.Unlock()
			go user.decreaseLimitTimeout()
		}()
		log.Println("New submition by "+b.users[ev.User].name, matches[0])
		youtubeID := videoID.FindStringSubmatch(matches[4])[2]
		entry, exist := b.entries[youtubeID]
		if exist {
			logInSlack("this video has already been submitted by "+b.users[entry.userID].name+": http://"+b.domain+entry.Path(), nil)
			return
		}
		if user.requestLimit < 2 {
			vid, err := ytdl.GetVideoInfo(youtubeID)
			if err != nil {
				logInSlack("Wrong YouTube ID", err)
				return
			}

			best := ytdl.Format{Extension: "empty"}
			for i := range vid.Formats {
				if vid.Formats[i].Extension == "mp4" {
					best = vid.Formats[i]
					break
				}
			}
			if best.Extension == "empty" {
				logInSlack("No mp4 found", nil)
				return
			}
			entry := newEntry(youtubeID, ev.User, time.Now())
			b.entries[youtubeID] = entry

			url, err := vid.GetDownloadURL(best)
			if err != nil {
				logInSlack("Could not get download URL of the video", err)
				return
			}

			out, err := exec.Command("bash", "-c", "ffmpeg -i \""+url.String()+"\" -f mp3 -vn "+entry.Path()).Output()
			if err != nil {
				logInSlack("error while converting video to mp3 "+string(out), err)
				return
			}
			b.logger(b.BTChannel.ID)(b.users[ev.User].name+" submitted a new challenge on http://"+b.domain+entry.Path(), nil)
		} else {
			_, _, channel, err := b.RTM.OpenIMChannel(ev.User)
			if err != nil {
				log.Println(err)
			} else {
				b.logger(channel)(strconv.Itoa(user.requestLimit)+" requests in a minute, slow down!", nil)
			}
		}
	}
}

func (u *user) decreaseLimitTimeout() {
	time.Sleep(time.Minute)
	u.Lock()
	u.requestLimit--
	u.Unlock()
}
