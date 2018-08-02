package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_shortAnswer(t *testing.T) {
	tests := []struct {
		answers string
		msg     string
		match   []string
		fail    []string
	}{
		{
			answers: "némo",
			msg:     "should match",
			match: []string{
				"le monde de nemo",
				"nemo",
				"n'emo",
				"le monde den emo",
				"Le Monde de Némo 2",
				"Néémo",
			},
			fail: []string{
				"Happy feet",
				"Mes monde de Nem",
			},
		},
		{
			answers: "monstre à Paris",
			msg:     "should match",
			match: []string{
				"Un Monstre a Paris",
				"Le Monstre à Paris",
				"monstreaparis",
			},
			fail: []string{
				"le monde de nemo",
				"nemo",
				"Le Monde de Némo 2",
			},
		},
		{
			answers: "prince d'égypte, Prince of Egypt",
			msg:     "should match",
			match: []string{
				"le prince d'egypte",
				"The prince of egypt",
				"prince dé'gypte",
			},
			fail: []string{
				"egypte",
				"egypt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.answers, func(t *testing.T) {
			for _, submitted := range tt.match {
				t.Run(submitted, func(t *testing.T) {
					assert.Equal(t, true, matchAnswers(submitted, tt.answers), tt.msg)
				})
			}
			for _, submitted := range tt.fail {
				t.Run(submitted, func(t *testing.T) {
					assert.Equal(t, false, matchAnswers(submitted, tt.answers), tt.msg)
				})
			}
		})
	}
}
