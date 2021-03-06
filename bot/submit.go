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
	httpRoot        = "https"
)

var (
	submission, _ = regexp.Compile(`^.*((http(s|):|)\/\/)?(www\.|)?yout(.*?)\/(embed\/|watch.*?v=|)([a-z_A-Z0-9\-]{11}).* "(.*)" "(.*)".*$`)
)

func (b *BlindBot) SubmitHandler(w http.ResponseWriter, r *http.Request) {
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
	submitterID := u.Query().Get("user_id")

	// submit submission
	_, exist := b.users[submitterID]
	if exist {
		go b.submitWithLogs(text, submitterID)
		w.WriteHeader(http.StatusCreated)
	} else {
		log.Println("UserID "+submitterID+" not found", err)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (b *BlindBot) submitWithLogs(text, submitterID string) {
	b.log(b.submit(text, submitterID), submitterID)
}

// submitterID MUST exist when calling this function
func (b *BlindBot) submit(text, submitterID string) error {
	matches := submission.FindStringSubmatch(text)
	if len(matches) == 0 {
		return fmt.Errorf("this submission does not follow the submission format")
	}
	log.Println("New submition by "+b.getUsername(submitterID), matches[0])

	// extract variables
	youtubeID := matches[7]
	answers := matches[8]
	hints := matches[9]
	user := b.users[submitterID]

	// apply rate limit
	user.increaseRateLimit()
	defer user.decreaseRateLimit()

	// check if this entry already exists
	entry, exist := b.getEntryFromYoutubeID(youtubeID)
	if exist {

		// the announcement failed previously
		if entry.threadID == "" {
			threadIDs, err := b.announce(b.AnnouncementMessage(hints, entry), b.blindTestChannelID)
			if err != nil {
				return err
			}
			return b.updateThread(entry, threadIDs[0])
		}

		// the youtubeID is missing
		if entry.youtubeID == "" {
			err := b.updateYoutubeID(entry, youtubeID)
			if err != nil {
				return err
			}
		}

		// the submitter update his answers
		if entry.winnerID == "" && (entry.submitterID == submitterID || submitterID == b.masterID) {
			return b.updateAnswers(entry, answers)
		}

		// already submitted
		return fmt.Errorf("this video has already been submitted by %s: %s://%s%s", b.getUsername(entry.submitterID), httpRoot, b.domain, entry.Path())
	}

	// check is user is rate limited
	if user.rateLimit >= submissionLimit {
		return fmt.Errorf("%s requests in a minute, slow down!", strconv.Itoa(user.rateLimit))
	}

	// create entry
	return b.createEntry(youtubeID, submitterID, answers, hints)
}

// download MP3 and create entry for the video
func (b *BlindBot) createEntry(youtubeID, submitterID, answers, hints string) error {
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
		return fmt.Errorf("cannot generate a download URL for this video %v", err)
	}

	// download mp3
	entry := newEntry(youtubeID, submitterID, answers, hints, time.Now())
	out, err := exec.Command("bash", "-c", "ffmpeg -i \""+url.String()+"\" -f mp3 -vn "+entry.Path()).Output()
	if err != nil {
		return fmt.Errorf("cannot convert video to mp3 %s %v", out, err)
	}

	// create new entry
	err = b.addEntry(entry)
	if err != nil {
		return err
	}

	threadIDs, err := b.announce(b.AnnouncementMessage(hints, entry), b.blindTestChannelID)
	if err != nil {
		return err
	}

	return b.updateThread(entry, threadIDs[0])
}
