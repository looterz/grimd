package main

import (
	"log"
	"time"
)

type ToggleData struct {
	Mode uint
	Data uint
}

type ActivationHandler struct {
	query_channel  chan bool
	toggle_channel chan ToggleData
	set_channel    chan bool
}

func (a *ActivationHandler) loop(quit <-chan bool) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var reactivate time.Time
	var reactivate_pending bool

	a.query_channel = make(chan bool)
	a.toggle_channel = make(chan ToggleData)
	a.set_channel = make(chan bool)

	ticker := time.Tick(1 * time.Second)

	var nextToggleTime = time.Now()

forever:
	for {
		select {
		case <-quit:
			break forever
		case <-a.query_channel:
			a.query_channel <- grimdActive
		case v := <-a.toggle_channel:
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
					reactivate_pending = true
				} else {
					reactivate_pending = false
				}
				a.query_channel <- grimdActive
			}
		case v := <-a.set_channel:
			grimdActive = v
			reactivate_pending = false
			a.set_channel <- grimdActive
		case <-ticker:
			now := time.Now()
			if reactivate_pending && now.After(reactivate) {
				log.Print("Reactivating grimd (timer)\n")
				grimdActive = true
				reactivate_pending = false
			}
		}
	}
}

// Query activation state
func (a ActivationHandler) query() bool {
	a.query_channel <- true
	return <-a.query_channel
}

// Set activation state
func (a ActivationHandler) set(v bool) bool {
	a.set_channel <- v
	return <-a.set_channel
}

// Toggle activation state on or off
func (a ActivationHandler) toggle() bool {
	data := ToggleData{
		Mode: 1,
		Data: Config.ReactivationDelay,
	}
	a.toggle_channel <- data
	return <-a.query_channel
}

// Like toggle(), but only from on to off. Toggling when off will restart the
// timer.
func (a ActivationHandler) toggleOff(timeout uint) bool {
	a.toggle_channel <- ToggleData{
		Mode: 2,
		Data: timeout,
	}
	return <-a.query_channel
}
