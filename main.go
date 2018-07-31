package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kindermoumoute/blindbot/bot"
)

var debug bool
var key string
var master string
var domain string
var botName string
var channel string

func init() {
	flag.BoolVar(&debug, "debug", false, "Set the debug mode")
	flag.StringVar(&key, "key", os.Getenv("SLACK_KEY"), "Set Slack API key")
	flag.StringVar(&master, "master", os.Getenv("SLACK_MASTER"), "Set Slack master user")
	flag.StringVar(&domain, "domain", os.Getenv("DOMAIN_NAME"), "Set server domain name")
	flag.StringVar(&botName, "name", "blindbot", "Set bot user name")
	flag.StringVar(&channel, "channel", "blindtest", "Set blind test channel")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	blindbot, err := bot.New(debug, key, master, domain, botName, channel)
	if err != nil {
		panic(err)
	}
	go runServer(blindbot)
	blindbot.Run()
}
