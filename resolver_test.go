package main

import (
	"testing"

	"github.com/miekg/dns"
)

func TestDoHLookup(t *testing.T) {
	resolver := Resolver{}
	_, err := resolver.DoHLookup("https://1.1.1.1/dns-query", 5, &dns.Msg{
		Question: []dns.Question{
			dns.Question{Name: dns.Fqdn("www.domain.com")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

}
