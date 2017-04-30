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
	timeSubs     map[int64][]specialTime // Mapping of people to an array of times
	timeSubsLock *sync.RWMutex           // Lock of this map
}

type config struct {
	Apikey string // Telegram API key
	Admins []int  // Bot admins
}

type specialTime struct {
	hours   uint8
	minutes uint8
	seconds uint8
}

type toBeSent struct {
	id   int64
	time specialTime
}

func (t specialTime) isEqualMinute(o specialTime) bool {
	return t.minutes == o.minutes && t.hours == o.hours
}
