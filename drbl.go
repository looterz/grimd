package main

import (
	"strings"
)

func drblCheckHostname(hostname string) bool {
	testhost := ""
	verdict := false
	if strings.HasSuffix(hostname, ".") {
		testhost = hostname[:len(hostname)-1]
		logger.Debug("a root query:", hostname)
	} else {
		testhost = hostname
		logger.Debug("not a root query:", hostname)
	}
	block, weight := drblPeers.Check(testhost)
	if block {
		verdict = true
		logger.Debug("DNS query:", testhost, "Got blocked with weigth", weight)
	}
	return verdict
}
