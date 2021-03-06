package main

import (
	"flag"
	"fmt"
	"os"

	"strings"

	"github.com/kindermoumoute/blindbot/bot"
)

var debug bool
var botUserKey string
var key string
var masterEmail string
var domains string
var botName string
var channel string
var dbPath string

func init() {
	flag.BoolVar(&debug, "debug", false, "Set the debug mode")
	flag.StringVar(&botUserKey, "botUserKey", os.Getenv("SLACK_KEY"), "Set Slack bot User Key")
	flag.StringVar(&key, "key", os.Getenv("SLACK_OAUTH2_KEY"), "Set Slack oauth2 API key")
	flag.StringVar(&masterEmail, "masterEmail", os.Getenv("SLACK_MASTER"), "Set Slack master email")
	flag.StringVar(&domains, "domains", os.Getenv("DOMAIN_NAMES"), "Set server domains name")
	flag.StringVar(&botName, "name", "blindbot", "Set bot user name")
	flag.StringVar(&channel, "channel", "blindtest", "Set blind test channel")
	flag.StringVar(&dbPath, "dbpath", "/db", "Set database directory")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	blindbot, err := bot.New(
		debug,
		botUserKey,
		key,
		masterEmail,
		botName,
		channel,
		dbPath,
		strings.Split(domains, ","),
	)
	if err != nil {
		panic(err)
	}
	go runServer(blindbot)
	blindbot.Run()
}
