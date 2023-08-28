package main

import "github.com/miekg/dns"

func NewCustomDNSRecordsFromText(recordsText []string) []CustomDNSRecords {
	customRecordsMap := make(map[string][]dns.RR)
	for _, recordText := range recordsText {
		answer, answerErr := dns.NewRR(recordText)
		if answerErr != nil {
			logger.Errorf("Cannot parse custom record: %s", answerErr)
		}
		name := answer.Header().Name
		if len(name) > 0 {
			if customRecordsMap[name] == nil {
				customRecordsMap[name] = []dns.RR{}
			}
			customRecordsMap[name] = append(customRecordsMap[name], answer)
		} else {
			logger.Errorf("Cannot parse custom record (invalid name): '%s'", recordText)
		}
	}
	return NewCustomDNSRecords(customRecordsMap)
}

func NewCustomDNSRecords(from map[string][]dns.RR) []CustomDNSRecords {
	var records []CustomDNSRecords
	for name, rrs := range from {
		records = append(records, CustomDNSRecords{
			name:   name,
			answer: rrs,
		})
	}
	return records
}

type CustomDNSRecords struct {
	name   string
	answer []dns.RR
}

func (c CustomDNSRecords) serve(server *DNSHandler) (handler func(dns.ResponseWriter, *dns.Msg)) {
	return func(writer dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)

		m.Answer = append(m.Answer, c.answer...)

		server.WriteReplyMsg(writer, m)
	}
}
