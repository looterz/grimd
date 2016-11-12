package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestCache(t *testing.T) {
	const (
		testDomain = "www.google.com"
	)

	cache := &MemoryCache{
		Backend:  make(map[string]Mesg, Config.Maxcount),
		Expire:   time.Duration(Config.Expire) * time.Second,
		Maxcount: Config.Maxcount,
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	if err := cache.Set(testDomain, m, true); err != nil {
		t.Error(err)
	}

	if _, _, err := cache.Get(testDomain); err != nil && err.Error() != fmt.Sprintf("%s expired", testDomain) {
		t.Error(err)
	}

	cache.Remove(testDomain)

	if _, _, err := cache.Get(testDomain); err == nil {
		t.Error("cache entry still existed after remove")
	}
}

func TestBlockCache(t *testing.T) {
	const (
		testDomain = "www.google.com"
	)

	cache := &MemoryBlockCache{
		Backend: make(map[string]bool),
	}

	if err := cache.Set(testDomain, true); err != nil {
		t.Error(err)
	}

	if exists := cache.Exists(testDomain); !exists {
		t.Error(testDomain, "didnt exist in block cache")
	}

	if _, err := cache.Get(testDomain); err != nil {
		t.Error(err)
	}

	if exists := cache.Exists(fmt.Sprintf("%sfuzz", testDomain)); exists {
		t.Error("fuzz existed in block cache")
	}
}
