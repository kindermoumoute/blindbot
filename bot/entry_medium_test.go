package bot

import (
	"os"
	"testing"
	"time"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/stretchr/testify/assert"
)

func TestEntries(t *testing.T) {
	testDBDir := "test/MyDatabase"
	os.RemoveAll(testDBDir)
	defer os.RemoveAll(testDBDir)

	// (Create if not exist) open a database
	testDB, err := db.OpenDB(testDBDir)
	if err != nil {
		panic(err)
	}
	assert.NoError(t, testDB.Create(EntryCollection))
	entriesDB := testDB.Use(EntryCollection)
	b := &BlindBot{
		entries: scanEntriesFromdb(entriesDB),
		db:      testDB,
	}
	// add an entry
	assert.Empty(t, b.entries)
	youtubeID := "oHg5SJYRHA0"
	entry := newEntry(youtubeID, "Flantier", "", time.Now())
	b.addEntry(entry)

	// update an entry
	entry, exist := b.getEntry(youtubeID)
	assert.True(t, exist)
	answers := "RickRoll'D"
	b.updateAnswers(entry, answers)
	assert.Equal(t, entry.answers, answers)

	// reload from db
	b.entries = scanEntriesFromdb(entriesDB)
	entry, exist = b.getEntry(youtubeID)
	assert.True(t, exist)
	assert.Equal(t, entry.answers, answers)
}
