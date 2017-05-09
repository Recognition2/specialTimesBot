package main

import "time"

func dataSaver() {
	defer logWarn.Println("Shutting data saver down")
	defer g.wg.Done()

outer:
	for {
		t := time.After(time.Minute * 2)
		select {
		case <-g.shutdown:
			Save("data.gob", g.timeSubs)
			break outer
		case <-t:
			logInfo.Println("Saving data...")
			Save("data.gob", g.timeSubs)
		}
	}
}
