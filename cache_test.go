package main

import (
	"fmt"
	"strings"
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

	if exists := cache.Exists(strings.ToUpper(testDomain)); !exists {
		t.Error(strings.ToUpper(testDomain), "didnt exist in block cache")
	}

	if _, err := cache.Get(testDomain); err != nil {
		t.Error(err)
	}

	if exists := cache.Exists(fmt.Sprintf("%sfuzz", testDomain)); exists {
		t.Error("fuzz existed in block cache")
	}
}

func TestBlockCacheGlob(t *testing.T) {
	const (
		globDomain1 = "*.google.com"
		globDomain2 = "ww?.google.com"
		testDomain1 = "www.google.com"
		testDomain2 = "wwx.google.com"
		testDomain3 = "www.google.it"
	)

	cache := &MemoryBlockCache{
		Backend: make(map[string]bool),
	}

	if err := cache.Set(globDomain1, true); err != nil {
		t.Error(err)
	}
	if err := cache.Set(globDomain2, true); err != nil {
		t.Error(err)
	}

	if exists := cache.Exists(testDomain1); !exists {
		t.Error(testDomain1, "didnt exist in block cache")
	}

	if exists := cache.Exists(testDomain2); !exists {
		t.Error(testDomain2, "didnt exist in block cache")
	}

	if exists := cache.Exists(testDomain3); exists {
		t.Error(testDomain3, "did exist in block cache")
	}
}
