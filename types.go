package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"sync"
)

type global struct {
	wg           *sync.WaitGroup         // For checking that everything has indeed shut down
	shutdown     chan bool               // To make sure everything can shut down
	bot          *tgbotapi.BotAPI        // The actual bot
	c            config                  // Configuration file
	timeSubs     map[int64][]SpecialTime // Mapping of people to an array of times
	timeSubsLock *sync.RWMutex           // Lock of this map
}

type config struct {
	Apikey string // Telegram API key
	Admins []int64  // Bot admins
}

type SpecialTime struct {
	Hours   uint8
	Minutes uint8
	Seconds uint8
}

type toBeSent struct {
	id   int64
	time SpecialTime
}

func (t SpecialTime) isEqualMinute(o SpecialTime) bool {
	return t.Minutes == o.Minutes && t.Hours == o.Hours
}
