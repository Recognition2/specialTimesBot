package main

import (
	"bytes"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"os/exec"
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

func sendSpecialTime(id int64, t SpecialTime) {
	text := "It's currently %d:%02d"
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

	if old.isEqualMinute(current) {
		// This is the same minute. Nothing can be done right now.
		return old
	}

	g.timeSubsLock.RLock()
	defer g.timeSubsLock.RUnlock()

	// Check all subscribed people whether they want to receive the current time
	for id, times := range g.timeSubs {
		// Iterate over all people
		for _, t := range times {
			// Iterate over all times per person
			//if t.Hours > current.Hours {
			// We're too far along in the array already
			//	break
			//}

			if !current.isEqualMinute(t) {
				// The current time is not special to this person
				continue
			}

			toSend <- toBeSent{id: id, time: current}
			break
		}
	}

	// Check whether we're taking too long
	if d := time.Since(cTime); d > 10*time.Second {
		logErr.Printf("Took %f seconds to loop through entire array ", d.Seconds())
		// Sending the bot admin a warning message will probably not help
		// if the bot is rate limited. However, try it anyway
		for _, v := range g.c.Admins {
			g.bot.Send(tgbotapi.NewMessage(v, "Checking all subscriptions took too long"))
		}
	}

	// Check whether admins want an update
	if current.Minutes%30 == 0 && current.Hours > 6 {
		// Send an update half an hour, except during the night
		uptime, err := exec.Command("uptime", "-p").Output()
		if err != nil {
			logErr.Println(err)
		}

		cmd := `grep MemAvail /proc/meminfo | awk '{print $2}' | xargs -I {} echo "scale=4; {}/1024" | bc`
		memAvail, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			logErr.Println(err)
		}

		cmd = "uptime | grep -ohe 'load average[s:][: ].*' | awk '{ print $3 }'"
		load, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			logErr.Println(err)
		}

		var txt bytes.Buffer
		txt.WriteString(fmt.Sprintf("Uptime: %sAvailable memory: %s MB\nCurrent load: %s", uptime[3:], memAvail[:len(load)-1], load[:len(load)-2]))
		for _, v := range g.c.Admins {
			g.bot.Send(tgbotapi.NewMessage(v, txt.String()))
		}

	}

	return current
}
