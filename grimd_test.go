package main

import (
	"testing"

	"github.com/miekg/dns"
)

const (
	testNameserver = "127.0.0.1:53"
	testDomain     = "www.google.com"
)

func BenchmarkResolver(b *testing.B) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	c := new(dns.Client)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Exchange(m, testNameserver)
	}
}
