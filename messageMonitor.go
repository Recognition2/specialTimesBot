package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"strconv"
	"strings"
)

func messageMonitor() {
	defer g.wg.Done()
	logInfo.Println("Starting message monitor")
	defer logWarn.Println("Stopping message monitor")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 300
	updates, err := g.bot.GetUpdatesChan(u)
	if err != nil {
		logErr.Printf("Update failed: %v\n", err)
	}

outer:
	for {
		select {
		case <-g.shutdown:
			break outer
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			if update.Message.IsCommand() {
				handleMessage(update.Message)
			}
		}
	}
}

func commandIsForMe(t string) bool {
	command := strings.SplitN(t, " ", 2)[0] // Return first substring before space, this is entire command

	i := strings.Index(command, "@") // Position of @ in command
	if i == -1 {                     // Not in command
		return true // Assume command is for everybody, including this bot
	}

	return strings.ToLower(command[i+1:]) == strings.ToLower(g.bot.Self.UserName)
}

func handleMessage(m *tgbotapi.Message) {
	if !commandIsForMe(m.Text) {
		return
	}

	switch strings.ToLower(m.Command()) {
	case "id":
		handleGetID(m)
	case "add":
		handleAddSpecialTime(m)
	case "remove":
		handleRemoveSpecialTime(m)
	case "clear":
		handleClearSpecialTime(m)
	case "start", "help":
		handleHelp(m)
	case "hi":
		handleHi(m)
	case "list":
		handleList(m)
	}
}

func handleList(m *tgbotapi.Message) {
	g.timeSubsLock.RLock()
	defer g.timeSubsLock.RUnlock()

	var b bytes.Buffer
	b.WriteString("This chat is currently subscribed to these times:\n")

	for _, j := range g.timeSubs[m.Chat.ID] {
		b.WriteString(fmt.Sprintf("%02d:%02d\n", j.Hours, j.Minutes))
	}
	g.bot.Send(tgbotapi.NewMessage(m.Chat.ID, b.String()))
}

func handleHelp(m *tgbotapi.Message) {
	msg := "This bot warns you at special times. Add a time at which you want to be warned every day using '/add'"
	g.bot.Send(tgbotapi.NewMessage(m.Chat.ID, msg))
}

func handleHi(m *tgbotapi.Message) {
	g.bot.Send(tgbotapi.NewMessage(m.Chat.ID, "Hi!"))

}

func handleAddSpecialTime(message *tgbotapi.Message) {
	g.timeSubsLock.Lock() // A time needs to be added, so lock the object for writing
	defer g.timeSubsLock.Unlock()

	sID := message.Chat.ID

	t, err := convToSpTime(message.CommandArguments())
	if err != nil {
		g.bot.Send(tgbotapi.NewMessage(sID, err.Error()))
		return
	}

	if spTimeExists(sID, t) != -1 {
		g.bot.Send(tgbotapi.NewMessage(sID, "Time already exists!"))
		return
	}

	g.timeSubs[message.Chat.ID] = append(g.timeSubs[sID], t)

	msg := tgbotapi.NewMessage(sID, "Time has been added successfully")
	msg.ReplyToMessageID = message.MessageID
	g.bot.Send(msg)
}

func spTimeExists(id int64, t SpecialTime) int {
	for k, v := range g.timeSubs[id] {
		if v.isEqualMinute(t) {
			return k
		}
	}
	return -1
}

func handleRemoveSpecialTime(message *tgbotapi.Message) {
	sID := message.Chat.ID
	g.timeSubsLock.Lock() // A time needs to be removed, so lock the object for writing
	defer g.timeSubsLock.Unlock()

	t, err := convToSpTime(message.CommandArguments())
	if err != nil {
		g.bot.Send(tgbotapi.NewMessage(sID, err.Error()))
		return
	}

	key := spTimeExists(sID, t)
	if key == -1 {
		g.bot.Send(tgbotapi.NewMessage(sID, "Time does not exist!"))
		return
	}

	tmp := g.timeSubs[sID]
	g.timeSubs[sID] = append(tmp[:key], tmp[key+1:]...)

	msg := tgbotapi.NewMessage(sID, "Time has been removed successfully")
	msg.ReplyToMessageID = message.MessageID
	g.bot.Send(msg)
}

func handleClearSpecialTime(message *tgbotapi.Message) {
	sID := message.Chat.ID

	g.timeSubsLock.Lock() // Removal of array, so lock the object for writing
	defer g.timeSubsLock.Unlock()

	delete(g.timeSubs, message.Chat.ID) // Delete the whole row of this chat from the map

	msg := tgbotapi.NewMessage(sID, "All times for this chat have been removed")
	msg.ReplyToMessageID = message.MessageID
	g.bot.Send(msg)
}

func convToSpTime(s string) (SpecialTime, error) {
	// This string must be of the format "21:30" or something
	arr := strings.Split(s, ":")
	if len(arr) > 3 {
		return SpecialTime{}, errors.New("Contains too many \":\"'s")
	}
	if len(arr) < 2 {
		return SpecialTime{}, errors.New("Time is missing. Please provide a time in the form '/add 22:35'")

	}

	hr, err := strconv.Atoi(arr[0])
	if err != nil {
		return SpecialTime{}, errors.New("This is not a number")
	}

	min, err := strconv.Atoi(arr[1])
	if err != nil {
		return SpecialTime{}, errors.New("This is not a number")
	}

	if hr > 23 || hr < 0 {
		return SpecialTime{}, errors.New("Not a valid hour")
	}

	if min > 59 || min < 0 {
		return SpecialTime{}, errors.New("Not a valid minute")
	}
	return SpecialTime{uint8(hr), uint8(min), 0}, nil
}

func handleGetID(cmd *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(cmd.Chat.ID, fmt.Sprintf("Hi, %s %s, your Telegram user ID is given by %d", cmd.From.FirstName, cmd.From.LastName, cmd.From.ID))
	_, err := g.bot.Send(msg)
	if err != nil {
		logErr.Println(err)
	}
}
