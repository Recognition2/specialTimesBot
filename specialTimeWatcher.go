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

	var old specialTime

outer:
	for {
		n := 10 * time.Second // Check whether people are to be notified every n seconds
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

func sendSpecialTime(id int64, t specialTime) {
	text := "Hi! It's currently %d:%d:%d, and I just wanted to make sure you knew, too!"
	msg := tgbotapi.NewMessage(id, fmt.Sprintf(text, t.hours, t.minutes, t.seconds))
	g.bot.Send(msg)
}

func checkSpecialTimes(old specialTime, toSend chan toBeSent) specialTime {
	// Create special time from the current time
	cTime := time.Now()
	var current specialTime
	current.hours = uint8(cTime.Hour())
	current.minutes = uint8(cTime.Minute())
	current.seconds = uint8(cTime.Second())

	g.timeSubsLock.RLock()
	defer g.timeSubsLock.RUnlock()

	for id, times := range g.timeSubs {
		if len(times) == 0 {
			// No subscriptions for this person
			continue
		}
		for _, t := range times {
			if !current.isEqualMinute(t) {
				// The current time is not special to this person
				continue
			}

			if current.seconds < 15 {
				// Skip this iteration, too close to 0 seconds
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
