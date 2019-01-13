package main

import (
	"log"
	"strings"
)

func drblCheckHostname(hostname string) bool {
	testhost := ""
	verdict := false
	if strings.HasSuffix(hostname, ".") {
		testhost = string(hostname[:len(hostname)-1])
		if Config.LogLevel > 0 {
			log.Println("a root query:", hostname)
		}
	} else {
		testhost = string(hostname)
		if Config.LogLevel > 0 {
			log.Println("not a root query:", hostname)
		}
	}
	block, weight := drblPeers.Check(testhost)
	if block {
		verdict = true
		if Config.LogLevel > 0 {
			log.Println("DNS query:", testhost, "Got blocked with weigth", weight)
		}
	}
	return verdict
}
