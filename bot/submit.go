package bot

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/rylio/ytdl"
)

const (
	submissionLimit = 5
	httpRoot        = "http"
)

var (
	submission, _ = regexp.Compile(`^.*((http(s|):|)\/\/)?(www\.|)?yout(.*?)\/(embed\/|watch.*?v=|)([a-z_A-Z0-9\-]{11}).* "(.*)" "(.*)".*$`)
)

func (b *Bot) SubmitHandler(w http.ResponseWriter, r *http.Request) {
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
	userID := u.Query().Get("user_id")

	// submit submission
	_, exist := b.users[userID]
	if exist {
		go b.submitWithLogs(text, userID)
		w.WriteHeader(http.StatusCreated)
	} else {
		log.Println("UserID "+userID+" not found", err)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (b *Bot) submitWithLogs(text, userID string) {
	b.log(b.submit(text, userID), userID)
}

// userID MUST exist when calling this function
func (b *Bot) submit(text, userID string) error {
	matches := submission.FindStringSubmatch(text)
	if len(matches) == 0 {
		return fmt.Errorf("this submission does not follow the submission format")
	}
	log.Println("New submition by "+b.getUsername(userID), matches[0])

	// extract variables
	youtubeID := matches[7]
	//answers := matches[8]	// unimplemented
	hints := matches[9]
	user := b.users[userID]

	// apply rate limit
	user.increaseRateLimit()
	defer user.decreaseRateLimit()

	// check if this entry already exists
	entry, exist := b.getEntry(youtubeID)
	if exist {
		return fmt.Errorf("this video has already been submitted by %s: %s://%s%s", b.getUsername(entry.userID), httpRoot, b.domain, entry.Path())
	}

	// check is user is rate limited
	if user.requestLimit >= submissionLimit {
		return fmt.Errorf("%s requests in a minute, slow down!", strconv.Itoa(user.requestLimit))
	}

	// create entry
	return b.createEntry(youtubeID, userID, hints)
}

// download MP3 and create entry for the video
func (b *Bot) createEntry(youtubeID, userID, hints string) error {
	b.rtm.SendMessage(b.rtm.NewTypingMessage(b.blindTestChannelID))

	// get video info
	vid, err := ytdl.GetVideoInfo(youtubeID)
	if err != nil {
		return fmt.Errorf("wrong youtube ID %v", err)
	}

	// find the best MP4 (contains MP3)
	best := ytdl.Format{Extension: "empty"}
	for i := range vid.Formats {
		if vid.Formats[i].Extension == "mp4" {
			best = vid.Formats[i]
			break
		}
	}
	if best.Extension == "empty" {
		return fmt.Errorf("no mp4 found")
	}

	// get video URL
	url, err := vid.GetDownloadURL(best)
	if err != nil {
		return fmt.Errorf("cannot genereate a download URL for this video %v", err)
	}

	// download mp3
	entry := newEntry(youtubeID, userID, time.Now())
	out, err := exec.Command("bash", "-c", "ffmpeg -i \""+url.String()+"\" -f mp3 -vn "+entry.Path()).Output()
	if err != nil {
		return fmt.Errorf("cannot convert video to mp3 %s %v", out, err)
	}

	// create new entry
	b.addEntry(entry)
	s := fmt.Sprintf("%s %s submitted a new challenge: %s://%s%s", hints, b.users[userID].name, httpRoot, b.domain, entry.Path())
	b.announce(s, b.blindTestChannelID)

	return nil
}
