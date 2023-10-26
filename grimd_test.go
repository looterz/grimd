package main

import (
	"github.com/BurntSushi/toml"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func BenchmarkResolver(b *testing.B) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	c := new(dns.Client)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := c.Exchange(m, testNameserver)
		if err != nil {
			logger.Error(err)
		}
	}
}

func TestMultipleARecords(t *testing.T) {
	var config Config
	_, err := toml.Decode(defaultConfig, &config)

	config.CustomDNSRecords = []string{
		"example.com.          IN  A       10.10.0.1 ",
		"example.com.          IN  A       10.10.0.2 ",
	}

	quitActivation := make(chan bool)
	actChannel := make(chan *ActivationHandler)

	go startActivation(actChannel, quitActivation, config.ReactivationDelay)
	grimdActivation = <-actChannel
	close(actChannel)

	server := &Server{
		host:     "0.0.0.0:53",
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}
	c := new(dns.Client)

	// BlockCache contains all blocked domains
	blockCache := &MemoryBlockCache{Backend: make(map[string]bool)}
	// QuestionCache contains all queries to the dns server
	questionCache := makeQuestionCache(config.QuestionCacheCap)

	server.Run(&config, blockCache, questionCache)
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)

	reply, _, err := c.Exchange(m, testNameserver)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if l := len(reply.Answer); l != 2 {
		t.Fatalf("Expected 2 returned records but had %v: %v", l, reply.Answer)
	}
}
