package bot

import (
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

var (
	entryCommand = regexp.MustCompile("^delete entry (.*)$")
)

func (b *BlindBot) masterCommands(ev *slack.MessageEvent) error {
	matches := entryCommand.FindStringSubmatch(ev.Text)
	if len(matches) != 0 {
		return b.deleteEntry(matches[1])
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
