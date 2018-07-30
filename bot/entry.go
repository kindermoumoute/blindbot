package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
)

const (
	rootPath = "/music/"
)

type entry struct {
	userID, hashedYoutubeID string
	submissionDate          time.Time
}

func newEntry(youtubeID, userID string, submissionDate time.Time) *entry {

	return &entry{
		userID:          userID,
		hashedYoutubeID: encryptYoutubeID(youtubeID),
		submissionDate:  submissionDate,
	}
}

func newEntryFromString(entry string) *entry {
	fields := strings.Split(strings.Split(entry, ".")[0], "-")
	if len(fields) != 3 {
		log.Printf("Could not create an entry from %s", entry)
		return nil
	}
	submittedTime, err := time.Parse("20060102150405", fields[2])
	if err != nil {
		log.Printf("Could not create an entry from %s, invalid time: %s", entry, fields[2])
		return nil
	}
	return newEntry(fields[0], fields[1], submittedTime)
}

func (e entry) String() string {
	return e.hashedYoutubeID + "-" + e.userID + e.submissionDate.Format("-20060102150405")
}

func (e entry) Path() string {
	return rootPath + e.String() + ".mp3"
}

func encryptYoutubeID(youtubeID string) string {
	newHasher, _ := blake2b.New512([]byte(youtubeID))
	return fmt.Sprintf("%x", newHasher.Sum([]byte(youtubeID))[:8])
}
