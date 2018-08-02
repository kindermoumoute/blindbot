package bot

import (
	"regexp"
	"strings"
	"unicode"

	"log"

	"github.com/nlopes/slack"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	digitsAndLetters, _ = regexp.Compile("[^a-zA-Z0-9]+")
)

func (b *BlindBot) validateAnswer(ev *slack.MessageEvent) error {
	log.Println(b.getUsername(ev.User) + " tried " + ev.Text)
	b.Lock()
	entry, exist := b.entriesByThreadID[ev.ThreadTimestamp]
	defer b.Unlock()
	if exist && entry.submitterID != ev.User && entry.winnerID == "" {
		if matchAnswers(ev.Text, entry.answers) {
			b.entries[entry.hashedYoutubeID].winnerID = ev.User
			err := b.writeClient.AddReaction("clap", slack.NewRefToMessage(ev.Channel, ev.Timestamp))
			if err != nil {
				return err
			}
			return b.writeClient.AddReaction("heavy_check_mark", slack.NewRefToMessage(ev.Channel, ev.ThreadTimestamp))
		}
		return b.writeClient.AddReaction("x", slack.NewRefToMessage(ev.Channel, ev.Timestamp))
	}
	return nil
}

func (b *BlindBot) newChallenge(ev *slack.MessageEvent) error {
	log.Println("new challenge detected " + ev.Text)
	matches := b.domainRegex.FindStringSubmatch(ev.Text)
	if len(matches) != 0 {
		b.Lock()
		// if deleting this function catch the ID in the link another way
		entry := newEntryFromString(matches[1], b.entries)
		b.Unlock()
		b.updateThread(entry, ev.Timestamp)
	}
	return nil
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func matchAnswers(submitted, expected string) bool {
	for _, answer := range strings.Split(expected, ",") {
		if strings.Contains(shortAnswer(submitted), shortAnswer(answer)) {
			return true
		}
	}
	return false
}

func shortAnswer(s string) string {
	b := make([]byte, len(s))

	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	_, _, e := t.Transform(b, []byte(s), true)
	if e != nil {
		panic(e)
	}
	processedString := digitsAndLetters.ReplaceAllString(string(b), "")
	return removeDuplicates(strings.ToLower(processedString))
}

func removeDuplicates(s string) string {
	result := []uint8{}
	slow := 0
	fast := 0
	for fast < len(s) {
		for fast < len(s) && s[slow] == s[fast] {
			fast++
		}
		result = append(result, s[slow])
		slow = fast
	}

	return string(result)
}
