package main

import (
	"flag"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/kindermoumoute/blindbot/bot"
)

var (
	collectionsDefault = []string{bot.EntryCollection, "players"}
)

var dbPath string

func init() {
	flag.StringVar(&dbPath, "dbpath", "/db", "Set database directory")
}

func initDB() *db.DB {
	// (Create if not exist) open a database
	myDB, err := db.OpenDB(dbPath)
	if err != nil {
		panic(err)
	}

	allCollections := myDB.AllCols()
	for _, collectionName := range collectionsDefault {
		exist := false
		for _, name := range allCollections {
			if name == collectionName {
				exist = true
			}
		}
		if !exist {
			if err := myDB.Create(collectionName); err != nil {
				panic(err)
			}
		}
	}

	return myDB
}
