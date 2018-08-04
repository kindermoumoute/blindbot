package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
)

const (
	EntryCollection = "entries"
)

var (
	rootPath             = "/music/"
	NoErrUpdatingAnswers = fmt.Errorf("Successfully updated answers. :+1:")
)

type entry struct {
	docID           int
	answers         string
	hashedYoutubeID string
	submitterID     string
	threadID        string
	youtubeID       string
	winnerID        string
	submissionDate  time.Time
}

// this function will be deprecated
func newEntryFromString(entryString string, entries map[string]*entry) *entry {
	fields := strings.Split(strings.Split(entryString, ".")[0], "-")
	if len(fields) != 3 {
		log.Printf("Could not create an entry from %s", entryString)
		return nil
	}

	_, exist := entries[fields[0]]
	if exist {
		return nil
	}

	submittedTime, err := time.Parse("20060102150405", fields[2])
	if err != nil {
		log.Printf("Could not create an entry from %s, invalid time: %s", entryString, fields[2])
		return nil
	}

	return &entry{
		hashedYoutubeID: fields[0],
		submitterID:     fields[1],
		submissionDate:  submittedTime,
	}
}

func (b *BlindBot) scanEntriesFromdb() {
	b.entries = make(map[string]*entry)
	b.db.Use(EntryCollection).ForEachDoc(func(id int, docContent []byte) (willMoveOn bool) {
		var entryDoc map[string]interface{}
		if json.Unmarshal(docContent, &entryDoc) != nil {
			log.Fatalln("cannot deserialize")
		}

		entry := &entry{
			docID:           id,
			answers:         entryDoc["answers"].(string),
			hashedYoutubeID: entryDoc["hashedYoutubeID"].(string),
			submitterID:     entryDoc["submitterID"].(string),
			threadID:        entryDoc["threadID"].(string),
			youtubeID:       entryDoc["youtubeID"].(string),
			winnerID:        entryDoc["winnerID"].(string),
		}

		entry.submissionDate, _ = time.Parse(time.RFC3339, entryDoc["submissionDate"].(string))

		b.entries[entry.hashedYoutubeID] = entry

		return true
	})
	b.syncEntries()
	log.Println(len(b.entries), "entries loaded")
}

func newEntry(youtubeID, submitterID, answers string, submissionDate time.Time) *entry {
	return &entry{
		hashedYoutubeID: encryptYoutubeID(youtubeID),
		youtubeID:       youtubeID,
		submitterID:     submitterID,
		submissionDate:  submissionDate,
		answers:         answers,
	}
}
func (b *BlindBot) syncEntries() {
	b.entriesByThreadID = make(map[string]*entry)
	for _, entry := range b.entries {
		if entry.threadID != "" {
			b.entriesByThreadID[entry.threadID] = entry
		} else {
			fmt.Println("no thread ID for ", entry.hashedYoutubeID)
		}
	}
}

func (b *BlindBot) addEntry(entry *entry) error {
	var err error
	b.Lock()
	firstEntry, exist := b.entries[entry.hashedYoutubeID]
	if exist {
		os.Remove(entry.Path())
		return fmt.Errorf("this video is being submitted by %s", b.getUsername(firstEntry.submitterID))
	}
	b.entries[entry.hashedYoutubeID] = entry
	b.Unlock()
	entry.docID, err = b.db.Use(EntryCollection).Insert(entry.toMap())
	return err
}

func (b *BlindBot) getEntryFromYoutubeID(youtubeID string) (entry *entry, exist bool) {
	b.Lock()
	entry, exist = b.entries[encryptYoutubeID(youtubeID)]
	b.Unlock()
	return
}

func (b *BlindBot) getEntry(hashedYoutubeID string) (entry *entry, exist bool) {
	b.Lock()
	entry, exist = b.entries[hashedYoutubeID]
	b.Unlock()
	return
}

func (e entry) toMap() map[string]interface{} {
	return map[string]interface{}{
		"answers":         e.answers,
		"hashedYoutubeID": e.hashedYoutubeID,
		"submitterID":     e.submitterID,
		"threadID":        e.threadID,
		"youtubeID":       e.youtubeID,
		"winnerID":        e.winnerID,
		"submissionDate":  e.submissionDate,
	}
}

func (b *BlindBot) updateEntry(entry *entry) error {
	return b.db.Use(EntryCollection).Update(entry.docID, entry.toMap())
}

func (b *BlindBot) updateAnswers(entry *entry, answers string) error {
	b.Lock()
	b.entries[entry.hashedYoutubeID].answers = answers
	b.Unlock()
	err := b.updateEntry(entry)
	if err == nil {
		err = NoErrUpdatingAnswers
	}
	return err
}

func (b *BlindBot) updateYoutubeID(entry *entry, youtubeID string) error {
	b.Lock()
	b.entries[entry.hashedYoutubeID].youtubeID = youtubeID
	b.Unlock()
	return b.updateEntry(entry)
}

func (b *BlindBot) updateWinner(entry *entry, winnerID string) error {
	b.Lock()
	b.entries[entry.hashedYoutubeID].winnerID = winnerID
	b.Unlock()
	return b.updateEntry(entry)
}

func (b *BlindBot) updateThread(entry *entry, threadID string) error {
	b.Lock()
	b.entries[entry.hashedYoutubeID].threadID = threadID
	b.entriesByThreadID[entry.threadID] = entry
	b.Unlock()
	return b.updateEntry(entry)
}

func (b *BlindBot) deleteEntry(hashedYoutubeID string) error {
	b.Lock()
	defer b.Unlock()

	entry, exist := b.entries[hashedYoutubeID]
	if !exist {
		return fmt.Errorf("no entry with this name")
	}

	// remove from memory cache
	_, exist = b.entriesByThreadID[entry.hashedYoutubeID]
	if exist {
		delete(b.entriesByThreadID, entry.hashedYoutubeID)
	}
	delete(b.entries, entry.hashedYoutubeID)

	// remove from db
	if err := b.db.Use(EntryCollection).Delete(entry.docID); err != nil {
		return err
	}
	// remove file
	return os.Remove(entry.Path())
}

func (e entry) String() string {
	return e.youtubeID + " " +
		e.hashedYoutubeID + " " +
		e.submitterID + e.submissionDate.Format(" 20060102150405 ") +
		e.answers + " " +
		e.winnerID + " " +
		e.threadID + " " +
		strconv.Itoa(e.docID)
}

func (e entry) Path() string {
	return rootPath + e.hashedYoutubeID + "-" + e.submitterID + e.submissionDate.Format("-20060102150405") + ".mp3"
}

func (b *BlindBot) AnnouncementMessage(hints string, entry *entry) string {
	return fmt.Sprintf("%s %s submitted a new challenge: %s://%s%s", hints, b.getUsername(entry.submitterID), httpRoot, b.domain, entry.Path())
}

func encryptYoutubeID(youtubeID string) string {
	newHasher, _ := blake2b.New512([]byte(youtubeID))
	return fmt.Sprintf("%x", newHasher.Sum([]byte(youtubeID))[:8])
}
