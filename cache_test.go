package main

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

const (
	testDomain     = "www.google.com"
	testNameserver = "127.0.0.1:53"
)

func TestCache(t *testing.T) {
	cache := makeCache()
	WallClock = clockwork.NewFakeClock()

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	if err := cache.Set(testDomain, m, true); err != nil {
		t.Error(err)
	}

	if _, _, err := cache.Get(testDomain); err != nil {
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

func TestCacheTtl(t *testing.T) {
	fakeClock := clockwork.NewFakeClock()
	WallClock = fakeClock
	cache := makeCache()

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	var attl uint32 = 10
	var aaaattl uint32 = 20
	nullroute := net.ParseIP("0.0.0.0")
	nullroutev6 := net.ParseIP("0::0")
	a := &dns.A{
		Hdr: dns.RR_Header{
			Name:   testDomain,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    attl,
		},
		A: nullroute}
	m.Answer = append(m.Answer, a)

	aaaa := &dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   testDomain,
			Rrtype: dns.TypeAAAA,
			Class:  dns.ClassINET,
			Ttl:    aaaattl,
		},
		AAAA: nullroutev6}
	m.Answer = append(m.Answer, aaaa)

	if err := cache.Set(testDomain, m, true); err != nil {
		t.Error(err)
	}

	msg, _, err := cache.Get(testDomain)
	assert.Nil(t, err)

	for _, answer := range msg.Answer {
		switch answer.Header().Rrtype {
		case dns.TypeA:
			assert.Equal(t, attl, answer.Header().Ttl, "TTL should be unchanged")
		case dns.TypeAAAA:
			assert.Equal(t, aaaattl, answer.Header().Ttl, "TTL should be unchanged")
		default:
			t.Error("Unexpected RR type")
		}
	}

	fakeClock.Advance(5 * time.Second)
	msg, _, err = cache.Get(testDomain)
	assert.Nil(t, err)

	for _, answer := range msg.Answer {
		switch answer.Header().Rrtype {
		case dns.TypeA:
			assert.Equal(t, attl-5, answer.Header().Ttl, "TTL should be decreased")
		case dns.TypeAAAA:
			assert.Equal(t, aaaattl-5, answer.Header().Ttl, "TTL should be decreased")
		default:
			t.Error("Unexpected RR type")
		}
	}

	fakeClock.Advance(5 * time.Second)
	_, _, err = cache.Get(testDomain)
	assert.Nil(t, err)

	for _, answer := range msg.Answer {
		switch answer.Header().Rrtype {
		case dns.TypeA:
			assert.Equal(t, uint32(0), answer.Header().Ttl, "TTL should be zero")
		case dns.TypeAAAA:
			assert.Equal(t, aaaattl-10, answer.Header().Ttl, "TTL should be decreased")
		default:
			t.Error("Unexpected RR type")
		}
	}

	fakeClock.Advance(1 * time.Second)

	// accessing an expired key will return KeyExpired error
	_, _, err = cache.Get(testDomain)
	if _, ok := err.(KeyExpired); !ok {
		t.Error(err)
	}

	// accessing an expired key will remove it from the cache
	_, _, err = cache.Get(testDomain)

	if _, ok := err.(KeyNotFound); !ok {
		t.Error("cache entry still existed after expiring - ", err)
	}
}

func TestCacheTtlFrequentPolling(t *testing.T) {
	const (
		testDomain = "www.google.com"
	)

	fakeClock := clockwork.NewFakeClock()
	WallClock = fakeClock
	cache := makeCache()

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	var attl uint32 = 10
	nullroute := net.ParseIP("0.0.0.0")
	a := &dns.A{
		Hdr: dns.RR_Header{
			Name:   testDomain,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    attl,
		},
		A: nullroute}
	m.Answer = append(m.Answer, a)

	if err := cache.Set(testDomain, m, true); err != nil {
		t.Error(err)
	}

	msg, _, err := cache.Get(testDomain)
	assert.Nil(t, err)

	assert.Equal(t, attl, msg.Answer[0].Header().Ttl, "TTL should be unchanged")

	//Poll 50 times at 100ms intervals: the TTL should go down by 5s
	for i := 0; i < 50; i++ {
		fakeClock.Advance(100 * time.Millisecond)
		_, _, err := cache.Get(testDomain)
		assert.Nil(t, err)
	}

	msg, _, err = cache.Get(testDomain)
	assert.Nil(t, err)

	assert.Equal(t, attl-5, msg.Answer[0].Header().Ttl, "TTL should be decreased")

	cache.Remove(testDomain)

}

func TestExpirationRace(t *testing.T) {
	cache := makeCache()
	fakeClock := clockwork.NewFakeClock()
	WallClock = fakeClock

	const testDomain = "www.domain.com"

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	nullroute := net.ParseIP("0.0.0.0")
	a := &dns.A{
		Hdr: dns.RR_Header{
			Name:   testDomain,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    1,
		},
		A: nullroute}
	m.Answer = append(m.Answer, a)

	//log.Printf("Adding %s with TTL %d\n", testDomain, attl)

	if err := cache.Set(testDomain, m, true); err != nil {
		t.Error(err)
	}

	for i := 0; i < 1000; i++ {
		fakeClock.Advance(time.Duration(100) * time.Millisecond)
		go cache.Get(testDomain)
		go cache.Set(testDomain, m, true)
	}
}

func addToCache(cache *MemoryQuestionCache, time int64) {
	cache.Add(QuestionCacheEntry{
		Date:    time,
		Remote:  fmt.Sprintf("%d", time),
		Blocked: true,
		Query:   Question{},
	})
}

func TestQuestionCacheGetFromTimestamp(t *testing.T) {
	memCache := makeQuestionCache(100)
	for i := 0; i < 100; i++ {
		addToCache(memCache, int64(i))
	}

	entries := memCache.GetOlder(50)
	assert.Len(t, entries, 49)
	entries = memCache.GetOlder(0)
	assert.Len(t, entries, 99)
	entries = memCache.GetOlder(-1)
	assert.Len(t, entries, 100)
	entries = memCache.GetOlder(200)
	assert.Len(t, entries, 0)
}

func BenchmarkSetCache(b *testing.B) {
	cache := makeCache()

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cache.Set(testDomain, m, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetCache(b *testing.B) {
	cache := makeCache()

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	if err := cache.Set(testDomain, m, false); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, _, err := cache.Get(testDomain); err != nil {
			b.Fatal(err)
		}
	}
}
