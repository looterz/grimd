package main

import (
	"time"
)

// ToggleData type
type ToggleData struct {
	Mode uint
	Data uint
}

// ActivationHandler type
type ActivationHandler struct {
	queryChannel  chan bool
	toggleChannel chan ToggleData
	setChannel    chan bool
}

func startActivation(actChannel chan *ActivationHandler, quit chan bool, reactivationDelay uint) {
	var reactivate time.Time
	var reactivatePending bool
	a := &ActivationHandler{}

	a.queryChannel = make(chan bool)
	a.toggleChannel = make(chan ToggleData)
	a.setChannel = make(chan bool)

	// put the reference to our struct in the channel
	// then continue to the loop
	actChannel <- a

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
				logger.Warning("Toggle is too close: wait 10 seconds\n")
			} else {
				if v.Mode == 1 {
					grimdActive = !grimdActive
				} else {
					grimdActive = false
				}
				nextToggleTime = time.Now().Add(time.Duration(10) * time.Second)
				if !grimdActive && reactivationDelay > 0 {
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
				logger.Notice("Reactivating grimd (timer)")
				grimdActive = true
				reactivatePending = false
			}
		}
	}
	logger.Debugf("Activation goroutine exiting")
	quit <- true
}

// Query activation state
func (a ActivationHandler) query() bool {
	a.queryChannel <- true
	return <-a.queryChannel
}

// Set activation state
func (a ActivationHandler) set(v bool) bool {
	a.setChannel <- v
	return <-a.setChannel
}

// Toggle activation state on or off
func (a ActivationHandler) toggle(reactivationDelay uint) bool {
	data := ToggleData{
		Mode: 1,
		Data: reactivationDelay,
	}
	a.toggleChannel <- data
	return <-a.queryChannel
}

// Like toggle(), but only from on to off. Toggling when off will restart the
// timer.
func (a ActivationHandler) toggleOff(timeout uint) bool {
	a.toggleChannel <- ToggleData{
		Mode: 2,
		Data: timeout,
	}
	return <-a.queryChannel
}
