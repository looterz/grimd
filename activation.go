package main

import (
    "time"
)

type ActivationHandler struct {
    query_channel chan bool
    toggle_channel chan bool
}

func (a* ActivationHandler) loop (quit <-chan bool){
    var reactivate time.Time
    var reactivate_pending bool

    a.query_channel = make(chan bool)
    a.toggle_channel = make(chan bool)

forever:
    for {
        select {
        case <- quit:
            break forever
        case <- a.query_channel:
            now := time.Now()
            if reactivate_pending && now.After(reactivate) {
                grimdActive = true
                reactivate_pending = false
            }
            a.query_channel <- grimdActive
            break
        case <- a.toggle_channel:
            grimdActive = !grimdActive
            if !grimdActive && Config.ReactivationDelay > 0 {
                reactivate = time.Now().Add(time.Duration(Config.ReactivationDelay) * time.Second)
                reactivate_pending = true
            } else {
                reactivate_pending = false
            }
            a.toggle_channel <- grimdActive
            break
        }
    }

}

func (a ActivationHandler) query() bool{
    a.query_channel <- true
    return <- a.query_channel
}

func (a ActivationHandler) toggle() bool {
    a.toggle_channel <- true
    return <- a.toggle_channel
}

