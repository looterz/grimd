package main

import "github.com/miekg/dns"

type CustomDNSRecord struct {
	handler *DNSHandler
	answer  dns.RR
}

func NewCustomDNSRecord(handler *DNSHandler, recordText string) (*CustomDNSRecord, error) {
	answer, answerErr := dns.NewRR(recordText)
	if answerErr != nil {
		return nil, answerErr
	}

	return &CustomDNSRecord{
		handler: handler,
		answer:  answer,
	}, nil
}

func (c *CustomDNSRecord) serve(writer dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(req)

	m.Answer = append(m.Answer, c.answer)

	c.handler.WriteReplyMsg(writer, m)
}
