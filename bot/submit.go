package bot

import (
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/rylio/ytdl"
)

const (
	submissionLimit = 3
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

func (b *Bot) youtubeURL(text, channel, userID string) {
	logInSlack := b.logger(channel)
	matches := youtubeURL.FindStringSubmatch(text)
	if len(matches) != 0 {
		user := b.users[userID]
		user.Lock()
		user.requestLimit++
		defer func() {
			time.Sleep(time.Second) // avoid the user to submit twice in a second
			user.Unlock()
			go user.decreaseLimitTimeout()
		}()
		log.Println("New submition by "+b.users[userID].name, matches[0])
		youtubeID := videoID.FindStringSubmatch(matches[4])[2]
		entry, exist := b.entries[encryptYoutubeID(youtubeID)]
		if exist {
			logInSlack("this video has already been submitted by "+b.users[entry.userID].name+": http://"+b.domain+entry.Path(), nil)
			return
		}
		if user.requestLimit < submissionLimit {
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
			entry := newEntry(youtubeID, userID, time.Now())
			b.entries[entry.hashedYoutubeID] = entry

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
			b.logger(b.BTChannel.ID)(b.users[userID].name+" submitted a new challenge on http://"+b.domain+entry.Path(), nil)
		} else {
			_, _, channel, err := b.RTM.OpenIMChannel(userID)
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
