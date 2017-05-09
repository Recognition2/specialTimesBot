package main

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"time"
)

func specialTimeWatcher() {
	defer g.wg.Done()
	defer logWarn.Println("Shutting time watcher down")

	g.wg.Add(1)
	toSend := make(chan toBeSent, 200) // Send queue can contain 200 messages at maximum
	go messageSender(toSend)

	var old SpecialTime

outer:
	for {
		n := 1 * time.Second // Check whether people are to be notified every n seconds
		t := time.After(n)
		select {
		case <-g.shutdown:
			break outer
		case <-t:
			old = checkSpecialTimes(old, toSend)
		}
	}
}
func messageSender(sents chan toBeSent) {
	defer g.wg.Done()
	defer logWarn.Println("Shutting down message sender")

outer:
	for {
		select {
		case <-g.shutdown:
			break outer
		case s := <-sents:
			sendSpecialTime(s.id, s.time)
		}
	}
}

func sendSpecialTime(id int64, t SpecialTime) {
	text := "Hi! It's currently %02d:%02d, and I just wanted to make sure you knew, too!"
	msg := tgbotapi.NewMessage(id, fmt.Sprintf(text, t.Hours, t.Minutes))
	g.bot.Send(msg)
}

func checkSpecialTimes(old SpecialTime, toSend chan toBeSent) SpecialTime {
	// Create special time from the current time
	cTime := time.Now()
	var current SpecialTime
	current.Hours = uint8(cTime.Hour())
	current.Minutes = uint8(cTime.Minute())
	current.Seconds = uint8(cTime.Second())

	if current.Seconds < 10 {
		// We're not enough into this minute yet, wait a little longer
		return old
	}

	g.timeSubsLock.RLock()
	defer g.timeSubsLock.RUnlock()

	for id, times := range g.timeSubs {
		if len(times) == 0 {
			// No subscriptions for this person
			println("No subscriptions")
			continue
		}
		for _, t := range times {
			if !current.isEqualMinute(t) {
				// The current time is not special to this person
				continue
			}

			if old.isEqualMinute(current) {
				// This message has already been sent
				continue
			}
			toSend <- toBeSent{id: id, time: current}
		}
	}
	return current
}
