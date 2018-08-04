package bot

import (
	"os"
	"testing"
	"time"

	"io/ioutil"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/stretchr/testify/assert"
)

func TestEntries(t *testing.T) {
	youtubeID := "oHg5SJYRHA0"
	submitterID := "Flantier"
	answers := "RickRoll'D,rickasley"
	threadID := "123456789"
	winner := "kindermoumoute"

	testDir := "test/"
	err := os.RemoveAll(testDir)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	// (Create if not exist) open a database
	testDB, err := db.OpenDB(testDir + "MyDatabase")
	if err != nil {
		panic(err)
	}
	assert.NoError(t, testDB.Create(EntryCollection))

	b := &BlindBot{
		entriesByThreadID: make(map[string]*entry),
		db:                testDB,
	}
	b.scanEntriesFromdb()

	// add an entry
	assert.Empty(t, b.entries)
	entry := newEntry(youtubeID, submitterID, "", time.Now())
	b.addEntry(entry)
	assert.Equal(t, entry.youtubeID, youtubeID)
	rootPath = testDir
	err = ioutil.WriteFile(entry.Path(), []byte("music-data"), 0644)
	assert.NoError(t, err)

	// get entry
	entry, exist := b.getEntry(youtubeID)
	assert.True(t, exist)

	// update answers
	err = b.updateAnswers(entry, answers)
	assert.EqualError(t, err, NoErrUpdatingAnswers.Error())
	assert.Equal(t, entry.answers, answers)

	// update threadID
	err = b.updateThread(entry, threadID)
	assert.NoError(t, err)
	assert.Equal(t, entry.threadID, threadID)
	b.syncEntries()
	assert.Len(t, b.entriesByThreadID, 1)

	// update winnerID
	err = b.updateWinner(entry, winner)
	assert.NoError(t, err)
	assert.Equal(t, entry.winnerID, winner)

	// reload from db
	b.scanEntriesFromdb()
	assert.Len(t, b.entries, 1)
	b.syncEntries()
	assert.Len(t, b.entriesByThreadID, 1)
	entry, exist = b.getEntry(youtubeID)
	assert.True(t, exist)
	assert.Equal(t, entry.answers, answers)
	assert.Equal(t, entry.threadID, threadID)
	assert.Equal(t, entry.winnerID, winner)
	assert.Equal(t, entry.submitterID, submitterID)
	assert.Equal(t, entry.youtubeID, youtubeID)
	assert.Equal(t, entry.hashedYoutubeID, encryptYoutubeID(youtubeID))

	// delete
	err = b.deleteEntry(entry.hashedYoutubeID)
	assert.NoError(t, err)
	assert.Len(t, b.entries, 0)

}
