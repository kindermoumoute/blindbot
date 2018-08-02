package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"log"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/kindermoumoute/blindbot/bot"
)

var (
	collectionsDefault = []string{bot.EntryCollection}
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

	for _, collectionName := range collectionsDefault {
		if !myDB.ColExists(collectionName) {
			log.Println("Create collection", collectionName)
			if err := myDB.Create(collectionName); err != nil {
				panic(err)
			}
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		<-sigs
		myDB.Close()
	}()

	return myDB
}