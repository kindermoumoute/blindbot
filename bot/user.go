package bot

import (
	"fmt"
	"sync"
	"time"
)

var (
	ErrMasterNotFound = fmt.Errorf("Master not found")
)

type user struct {
	sync.Mutex
	name               string
	requestVeilleCount int
	requestLimit       int
	channelID          string
}

func (b *Bot) scanUsers(masterEmail, botName string) error {
	if b.users == nil {
		b.users = make(map[string]*user)
	}

	users, err := b.client.GetUsers()
	if err != nil {
		return err
	}

	err = ErrMasterNotFound
	for _, u := range users {
		if u.Profile.Email == masterEmail {
			b.masterID = u.ID
			err = nil
		}
		if u.IsBot && u.Name == botName {
			b.id = u.ID
		}
		b.users[u.ID] = &user{
			name: u.Name,
		}
	}
	return err
}

func (b *Bot) getUsername(userID string) string {
	return b.users[userID].name
}

func (u *user) increaseRateLimit() {
	u.Lock()
	u.requestLimit++
}

func (u *user) decreaseRateLimit() {
	time.Sleep(time.Second) // avoid the user to submit twice in a second
	u.Unlock()
	go func() {
		time.Sleep(time.Minute)
		u.Lock()
		u.requestLimit--
		u.Unlock()
	}()
}