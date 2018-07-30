package bot

import (
	"log"
	"strings"
	"time"
)

const (
	rootPath = "/music/"
)

type entry struct {
	userID, youtubeID string
	submissionDate    time.Time
}

func newEntry(youtubeID, userID string, submissionDate time.Time) *entry {
	return &entry{
		userID:         userID,
		youtubeID:      youtubeID,
		submissionDate: submissionDate,
	}
}

func newEntryFromString(entry string) (string, *entry) {
	fields := strings.Split(strings.Split(entry, ".")[0], "-")
	if len(fields) != 3 {
		log.Printf("Could not create an entry from %s", entry)
		return "", nil
	}
	submittedTime, err := time.Parse(
		"20060102150405",
		fields[2])
	if err != nil {
		log.Printf("Could not create an entry from %s, invalid time: %s", entry, fields[2])
		return fields[0], nil
	}
	return fields[0], newEntry(fields[0], fields[1], submittedTime)
}

func (e entry) String() string {
	return e.youtubeID + "-" + e.userID + e.submissionDate.Format("-20060102150405")
}

func (e entry) Path() string {
	return rootPath + e.String() + ".mp3"
}
