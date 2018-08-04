package bot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

var (
	entryCommand = regexp.MustCompile("^delete entry (.*)$")
	updateWinner = regexp.MustCompile("^update winner (.*) (.*)$")
)

func (b *BlindBot) masterCommands(ev *slack.MessageEvent) error {
	matches := entryCommand.FindStringSubmatch(ev.Text)
	if len(matches) != 0 {
		return b.deleteEntry(matches[1])
	}
	matches = updateWinner.FindStringSubmatch(ev.Text)
	if len(matches) != 0 {
		entry, exist := b.getEntry(matches[1])
		if exist {
			return b.updateWinner(entry, matches[2])
		}
		return fmt.Errorf("%s is not a valid entry", matches[1])
	}
	if strings.Contains(ev.Text, "show entries") {
		b.Lock()
		message := ""
		count := 0
		for _, entry := range b.entries {
			message += entry.String() + "\n"
			count++
			if count%50 == 0 {
				b.announce(message, b.masterChannelID())
				message = ""
			}
		}
		b.announce(message, b.masterChannelID())
		b.Unlock()
	}

	return nil
}
