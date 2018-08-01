package bot

import (
	"fmt"
	"io/ioutil"
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

func scanEntries() map[string]*entry {
	e := make(map[string]*entry)
	files, err := ioutil.ReadDir(rootPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		entry := newEntryFromString(f.Name())
		if entry != nil {
			e[entry.hashedYoutubeID] = entry
		}
	}
	return e
}

func newEntry(youtubeID, userID string, submissionDate time.Time) *entry {
	return &entry{
		hashedYoutubeID: encryptYoutubeID(youtubeID),
		userID:          userID,
		submissionDate:  submissionDate,
	}
}

func (b *Bot) addEntry(entry *entry) {
	b.Lock()
	b.entries[entry.hashedYoutubeID] = entry
	b.Unlock()
}

func newEntryFromString(entryString string) *entry {
	fields := strings.Split(strings.Split(entryString, ".")[0], "-")
	if len(fields) != 3 {
		log.Printf("Could not create an entry from %s", entryString)
		return nil
	}
	submittedTime, err := time.Parse("20060102150405", fields[2])
	if err != nil {
		log.Printf("Could not create an entry from %s, invalid time: %s", entryString, fields[2])
		return nil
	}
	return &entry{
		hashedYoutubeID: fields[0],
		userID:          fields[1],
		submissionDate:  submittedTime,
	}
}

func (b *Bot) getEntry(youtubeID string) (entry *entry, exist bool) {
	b.Lock()
	entry, exist = b.entries[encryptYoutubeID(youtubeID)]
	b.Unlock()
	return
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
