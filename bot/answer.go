package bot

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	digitsAndLetters, _ = regexp.Compile("[^a-zA-Z0-9]+")
)

func (b *BlindBot) updateAnswers(entry *entry, answers string) error {
	b.Lock()
	b.entries[entry.hashedYoutubeID].answers = answers
	id := entry.docID
	b.Unlock()
	err := b.db.Use(EntryCollection).Update(id, entry.toMap())
	if err == nil {
		err = fmt.Errorf("Successfully updated answers. :+1:")
	}
	return err
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}
func matchAnswers(submitted, expected string) bool {
	for _, answer := range strings.Split(expected, ",") {
		if strings.Contains(shortAnswer(submitted), shortAnswer(answer)) {
			return true
		}
	}
	return false
}

func shortAnswer(s string) string {
	b := make([]byte, len(s))

	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	_, _, e := t.Transform(b, []byte(s), true)
	if e != nil {
		panic(e)
	}
	processedString := digitsAndLetters.ReplaceAllString(string(b), "")
	return removeDuplicates(strings.ToLower(processedString))
}

func removeDuplicates(s string) string {
	result := []uint8{}
	slow := 0
	fast := 0
	for fast < len(s) {
		for fast < len(s) && s[slow] == s[fast] {
			fast++
		}
		result = append(result, s[slow])
		slow = fast
	}

	return string(result)
}
