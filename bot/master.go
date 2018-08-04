package bot

import (
	"fmt"
	"regexp"
	"strings"

	"log"

	"github.com/nlopes/slack"
)

var (
	entryCommand = regexp.MustCompile(`^delete entry (.*)$`)
	updateWinner = regexp.MustCompile(`^update entry (.*) (.*) "(.*)" "(.*)"$`)
)

func (b *BlindBot) masterCommands(ev *slack.MessageEvent) error {
	matches := entryCommand.FindStringSubmatch(ev.Text)
	if len(matches) != 0 {
		log.Println("Deleting ", matches[1])
		return b.deleteEntry(matches[1])
	}
	matches = updateWinner.FindStringSubmatch(ev.Text)
	if len(matches) != 0 {
		entry, exist := b.getEntry(matches[2])
		if exist {
			log.Println("Updating ", matches[2])
			err := b.updateWinner(entry, matches[1])
			if err != nil {
				return err
			}
			err = b.updateHints(entry, matches[4])
			if err != nil {
				return err
			}
			return b.updateAnswers(entry, matches[3])
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
