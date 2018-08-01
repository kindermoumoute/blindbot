package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/HouzuoGuo/tiedot/db"
	"golang.org/x/crypto/blake2b"
)

const (
	rootPath        = "/music/"
	EntryCollection = "entries"
)

type entry struct {
	submitterID, hashedYoutubeID, winnerID, answers string
	submissionDate                                  time.Time
	docID                                           int
}

func scanEntries(db *db.DB) map[string]*entry {
	entriesDB := db.Use(EntryCollection)
	e := scanEntriesFromdb(entriesDB)

	// scanning the rootPath will be deprecated
	files, err := ioutil.ReadDir(rootPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		entry := newEntryFromString(f.Name(), e)
		if entry != nil {
			e[entry.hashedYoutubeID] = entry
			log.Println("migrating " + entry.hashedYoutubeID + " to database")
			e[entry.hashedYoutubeID].docID, err = entriesDB.Insert(map[string]interface{}{
				"submitterID":    entry.submitterID,
				"youtubeID":      entry.hashedYoutubeID,
				"submissionDate": entry.submissionDate,
			})
			if err != nil {
				log.Println(err)
			}
		}
	}
	return e
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

func scanEntriesFromdb(entriesDB *db.Col) map[string]*entry {
	entriesMap := make(map[string]*entry)
	entriesDB.ForEachDoc(func(id int, docContent []byte) (willMoveOn bool) {
		var entryDoc map[string]interface{}
		if json.Unmarshal(docContent, &entryDoc) != nil {
			log.Fatalln("cannot deserialize")
		}
		entry := &entry{
			hashedYoutubeID: entryDoc["submitterID"].(string),
			submitterID:     entryDoc["youtubeID"].(string),
			submissionDate:  entryDoc["submissionDate"].(time.Time),
			docID:           id,
		}

		winner, exist := entryDoc["winner"].(string)
		if exist {
			entry.winnerID = winner
		}

		answers, exist := entryDoc["answers"].(string)
		if exist {
			entry.answers = answers
		}

		entriesMap[entry.hashedYoutubeID] = entry

		return true
	})
	return entriesMap
}

func newEntry(youtubeID, submitterID, answers string, submissionDate time.Time) *entry {
	return &entry{
		hashedYoutubeID: encryptYoutubeID(youtubeID),
		submitterID:     submitterID,
		submissionDate:  submissionDate,
		answers:         answers,
	}
}

func (b *BlindBot) addEntry(entry *entry) {
	var err error
	b.Lock()
	b.entries[entry.hashedYoutubeID] = entry
	b.Unlock()
	entry.docID, err = b.db.Use(EntryCollection).Insert(map[string]interface{}{
		"submitterID":    entry.submitterID,
		"youtubeID":      entry.hashedYoutubeID,
		"submissionDate": entry.submissionDate,
		"answers":        entry.answers,
	})
	if err != nil {
		b.log(err)
	}
}

func (b *BlindBot) getEntry(youtubeID string) (entry *entry, exist bool) {
	b.Lock()
	entry, exist = b.entries[encryptYoutubeID(youtubeID)]
	b.Unlock()
	return
}

func (e entry) String() string {
	return e.hashedYoutubeID + " " + e.submitterID + e.submissionDate.Format(" 20060102150405 ") + e.answers + " " + strconv.Itoa(e.docID)
}

func (e entry) Path() string {
	return rootPath + e.hashedYoutubeID + "-" + e.submitterID + e.submissionDate.Format("-20060102150405") + ".mp3"
}

func encryptYoutubeID(youtubeID string) string {
	newHasher, _ := blake2b.New512([]byte(youtubeID))
	return fmt.Sprintf("%x", newHasher.Sum([]byte(youtubeID))[:8])
}
