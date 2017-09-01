package main

import (
	"log"
	"time"
)

// toggleData
type toggleData struct {
	Mode uint
	Data uint
}

// activationHandler
type activationHandler struct {
	queryChannel  chan bool
	toggleChannel chan toggleData
	setChannel    chan bool
}

func (a *activationHandler) loop(quit <-chan bool) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var reactivate time.Time
	var reactivatePending bool

	a.queryChannel = make(chan bool)
	a.toggleChannel = make(chan toggleData)
	a.setChannel = make(chan bool)

	ticker := time.Tick(1 * time.Second)

	var nextToggleTime = time.Now()

forever:
	for {
		select {
		case <-quit:
			break forever
		case <-a.queryChannel:
			a.queryChannel <- grimdActive
		case v := <-a.toggleChannel:
			// Firefox is sending 2 queries in a row, so debouncing is needed.
			if v.Mode == 1 && nextToggleTime.After(time.Now()) {
				log.Print("Toggle is too close: wait 10 seconds\n")
			} else {
				if v.Mode == 1 {
					grimdActive = !grimdActive
				} else {
					grimdActive = false
				}
				nextToggleTime = time.Now().Add(time.Duration(10) * time.Second)
				if !grimdActive && Config.ReactivationDelay > 0 {
					reactivate = time.Now().Add(time.Duration(v.Data) * time.Second)
					reactivatePending = true
				} else {
					reactivatePending = false
				}
				a.queryChannel <- grimdActive
			}
		case v := <-a.setChannel:
			grimdActive = v
			reactivatePending = false
			a.setChannel <- grimdActive
		case <-ticker:
			now := time.Now()
			if reactivatePending && now.After(reactivate) {
				log.Print("Reactivating grimd (timer)\n")
				grimdActive = true
				reactivatePending = false
			}
		}
	}
}

// Query activation state
func (a activationHandler) query() bool {
	a.queryChannel <- true
	return <-a.queryChannel
}

// Set activation state
func (a activationHandler) set(v bool) bool {
	a.setChannel <- v
	return <-a.setChannel
}

// Toggle activation state on or off
func (a activationHandler) toggle() bool {
	data := toggleData{
		Mode: 1,
		Data: Config.ReactivationDelay,
	}
	a.toggleChannel <- data
	return <-a.queryChannel
}

// Like toggle(), but only from on to off. Toggling when off will restart the
// timer.
func (a activationHandler) toggleOff(timeout uint) bool {
	a.toggleChannel <- toggleData{
		Mode: 2,
		Data: timeout,
	}
	return <-a.queryChannel
}
