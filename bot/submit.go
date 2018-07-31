package bot

import (
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"io/ioutil"

	"github.com/rylio/ytdl"
)

const (
	submissionLimit = 3
)

var (
	youtubeURL, _ = regexp.Compile(`.<*(https?\:\/\/)?(www\.)?(youtube\.com|youtu\.?be)\/(.+)>.*`)
	videoID, _    = regexp.Compile(`^(watch\?)?(.*)$`)
	submission, _ = regexp.Compile(`^"(.*)" "(.*)" "(.*)"$`)
)

func (b *Bot) Submit(w http.ResponseWriter, r *http.Request) {
	// read body
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
	}

	// extract body parameters
	u, err := url.ParseRequestURI("/?" + string(bodyBytes))
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
	}
	text := u.Query().Get("text")
	channelID := u.Query().Get("channel_id")
	userID := u.Query().Get("user_id")

	// extract command parameters
	matches := submission.FindStringSubmatch(text)
	if len(matches) != 4 {
		log.Println(matches, text, string(bodyBytes))
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	// submit submission
	_, exist := b.users[userID]
	if exist {
		w.WriteHeader(http.StatusCreated)
		go b.youtubeURL(matches[1], channelID, userID, matches[3])
	} else {
		log.Println("UserID " + userID + " not found")
		w.WriteHeader(http.StatusNotFound)
	}
}

func (b *Bot) youtubeURL(urlText, channelID, userID, hints string) {
	logInSlack := b.logger(channelID)
	matches := youtubeURL.FindStringSubmatch(urlText)
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
		IDMatches := videoID.FindStringSubmatch(matches[4])
		if IDMatches[1] == "" {
			IDMatches[2] = "v=" + IDMatches[2]
		}
		u, err := url.ParseRequestURI("/?" + IDMatches[2])
		if err != nil {
			logInSlack("wrong URI", err)
			return
		}
		youtubeID := u.Query().Get("v")
		entry, exist := b.entries[encryptYoutubeID(youtubeID)]
		if exist {
			logInSlack("this video has already been submitted by "+b.users[entry.userID].name+": http://"+b.domain+entry.Path(), nil)
			return
		}
		if user.requestLimit < submissionLimit {
			b.RTM.SendMessage(b.RTM.NewTypingMessage(b.BTChannel.ID))
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
			b.logger(b.BTChannel.ID)(hints+" "+b.users[userID].name+" submitted a new challenge: http://"+b.domain+entry.Path(), nil)
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

func (u *user) decreaseLimitTimeout() {
	time.Sleep(time.Minute)
	u.Lock()
	u.requestLimit--
	u.Unlock()
}
