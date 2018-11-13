package main

import (
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	notIPQuery = 0
	_IP4Query  = 4
	_IP6Query  = 6
)

// Question type
type Question struct {
	Qname  string `json:"name"`
	Qtype  string `json:"type"`
	Qclass string `json:"class"`
}

// QuestionCacheEntry represents a full query from a client with metadata
type QuestionCacheEntry struct {
	Date    int64    `json:"date"`
	Remote  string   `json:"client"`
	Blocked bool     `json:"blocked"`
	Query   Question `json:"query"`
}

// String formats a question
func (q *Question) String() string {
	return q.Qname + " " + q.Qclass + " " + q.Qtype
}

// DNSHandler type
type DNSHandler struct {
	requestChannel chan DNSOperationData
	resolver       *Resolver
	cache          Cache
	negCache       Cache
}

// DNSOperationData type
type DNSOperationData struct {
	Net string
	w   dns.ResponseWriter
	req *dns.Msg
}

// NewHandler returns a new DNSHandler
func NewHandler(config *Config, blockCache *MemoryBlockCache, questionCache *MemoryQuestionCache) *DNSHandler {
	var (
		clientConfig *dns.ClientConfig
		resolver     *Resolver
		cache        Cache
		negCache     Cache
	)

	resolver = &Resolver{clientConfig}

	cache = &MemoryCache{
		Backend:  make(map[string]*Mesg, config.Maxcount),
		Maxcount: config.Maxcount,
	}
	negCache = &MemoryCache{
		Backend:  make(map[string]*Mesg),
		Maxcount: config.Maxcount,
	}

	handler := &DNSHandler{
		requestChannel: make(chan DNSOperationData),
		resolver:       resolver,
		cache:          cache,
		negCache:       negCache,
	}

	go handler.do(config, blockCache, questionCache)

	return handler
}

func (h *DNSHandler) do(config *Config, blockCache *MemoryBlockCache, questionCache *MemoryQuestionCache) {
	for {
		data, ok := <-h.requestChannel
		if !ok {
			break
		}
		func(Net string, w dns.ResponseWriter, req *dns.Msg) {
			defer w.Close()
			q := req.Question[0]
			Q := Question{UnFqdn(q.Name), dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass]}

			var remote net.IP
			if Net == "tcp" {
				remote = w.RemoteAddr().(*net.TCPAddr).IP
			} else {
				remote = w.RemoteAddr().(*net.UDPAddr).IP
			}

			logger.Infof("%s lookup　%s\n", remote, Q.String())

			var grimdActive = grimdActivation.query()
			if len(config.ToggleName) > 0 && strings.Contains(Q.Qname, config.ToggleName) {
				logger.Noticef("Found ToggleName! (%s)\n", Q.Qname)
				grimdActive = grimdActivation.toggle(config.ReactivationDelay)

				if grimdActive {
					logger.Notice("Grimd Activated")
				} else {
					logger.Notice("Grimd Deactivated")
				}
			}

			IPQuery := h.isIPQuery(q)

			// Only query cache when qtype == 'A'|'AAAA' , qclass == 'IN'
			key := KeyGen(Q)
			if IPQuery > 0 {
				mesg, blocked, err := h.cache.Get(key)
				if err != nil {
					if mesg, blocked, err = h.negCache.Get(key); err != nil {
						logger.Debugf("%s didn't hit cache\n", Q.String())
					} else {
						logger.Debugf("%s hit negative cache\n", Q.String())
						h.HandleFailed(w, req)
						return
					}
				} else {
					if blocked && !grimdActive {
						logger.Debugf("%s hit cache and was blocked: forwarding request\n", Q.String())
					} else {
						logger.Debugf("%s hit cache\n", Q.String())

						// we need this copy against concurrent modification of Id
						msg := *mesg
						msg.Id = req.Id
						h.WriteReplyMsg(w, &msg)
						return
					}
				}
			}
			// Check blocklist
			var blacklisted = false

			if IPQuery > 0 {
				blacklisted = blockCache.Exists(Q.Qname)

				if grimdActive && blacklisted {
					m := new(dns.Msg)
					m.SetReply(req)

					nullroute := net.ParseIP(config.Nullroute)
					nullroutev6 := net.ParseIP(config.Nullroutev6)

					switch IPQuery {
					case _IP4Query:
						rrHeader := dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    config.TTL,
						}
						a := &dns.A{Hdr: rrHeader, A: nullroute}
						m.Answer = append(m.Answer, a)
					case _IP6Query:
						rrHeader := dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeAAAA,
							Class:  dns.ClassINET,
							Ttl:    config.TTL,
						}
						a := &dns.AAAA{Hdr: rrHeader, AAAA: nullroutev6}
						m.Answer = append(m.Answer, a)
					}

					h.WriteReplyMsg(w, m)

					logger.Noticef("%s found in blocklist\n", Q.Qname)

					// log query
					NewEntry := QuestionCacheEntry{Date: time.Now().Unix(), Remote: remote.String(), Query: Q, Blocked: true}
					go questionCache.Add(NewEntry)

					// cache the block; we don't know the true TTL for blocked entries: we just enforce our config
					err := h.cache.Set(key, m, true)
					if err != nil {
						logger.Errorf("Set %s block cache failed: %s\n", Q.String(), err.Error())
					}

					return
				}
				logger.Debugf("%s not found in blocklist\n", Q.Qname)
			}

			// log query
			NewEntry := QuestionCacheEntry{Date: time.Now().Unix(), Remote: remote.String(), Query: Q, Blocked: false}
			go questionCache.Add(NewEntry)

			mesg, err := h.resolver.Lookup(Net, req, config.Timeout, config.Interval, config.Nameservers, config.DoH)

			if err != nil {
				logger.Errorf("resolve query error %s\n", err)
				h.HandleFailed(w, req)

				// cache the failure, too!
				if err = h.negCache.Set(key, nil, false); err != nil {
					logger.Errorf("set %s negative cache failed: %v\n", Q.String(), err)
				}
				return
			}

			if mesg.Truncated && Net == "udp" {
				mesg, err = h.resolver.Lookup("tcp", req, config.Timeout, config.Interval, config.Nameservers, config.DoH)
				if err != nil {
					logger.Errorf("resolve tcp query error %s\n", err)
					h.HandleFailed(w, req)

					// cache the failure, too!
					if err = h.negCache.Set(key, nil, false); err != nil {
						logger.Errorf("set %s negative cache failed: %v\n", Q.String(), err)
					}
					return
				}
			}

			//find the smallest ttl
			ttl := config.Expire
			var candidateTTL uint32

			for index, answer := range mesg.Answer {
				logger.Debugf("Answer %d - %s\n", index, answer.String())

				candidateTTL = answer.Header().Ttl

				if candidateTTL > 0 && candidateTTL < ttl {
					ttl = candidateTTL
				}
			}

			h.WriteReplyMsg(w, mesg)

			if IPQuery > 0 && len(mesg.Answer) > 0 {
				if !grimdActive && blacklisted {
					logger.Debugf("%s is blacklisted and grimd not active: not caching\n", Q.String())
				} else {
					err = h.cache.Set(key, mesg, false)
					if err != nil {
						logger.Errorf("set %s cache failed: %s\n", Q.String(), err.Error())
					}
					logger.Debugf("insert %s into cache with ttl %d\n", Q.String(), ttl)
				}
			}
		}(data.Net, data.w, data.req)
	}
}

// DoTCP begins a tcp query
func (h *DNSHandler) DoTCP(w dns.ResponseWriter, req *dns.Msg) {
	h.requestChannel <- DNSOperationData{"tcp", w, req}
}

// DoUDP begins a udp query
func (h *DNSHandler) DoUDP(w dns.ResponseWriter, req *dns.Msg) {
	h.requestChannel <- DNSOperationData{"udp", w, req}
}

// HandleFailed handles dns failures
func (h *DNSHandler) HandleFailed(w dns.ResponseWriter, message *dns.Msg) {
	m := new(dns.Msg)
	m.SetRcode(message, dns.RcodeServerFailure)
	h.WriteReplyMsg(w, m)
}

// WriteReplyMsg writes the dns reply
func (h *DNSHandler) WriteReplyMsg(w dns.ResponseWriter, message *dns.Msg) {
	defer func() {
		if r := recover(); r != nil {
			logger.Noticef("Recovered in WriteReplyMsg: %s\n", r)
		}
	}()

	err := w.WriteMsg(message)
	if err != nil {
		logger.Error(err)
	}
}

func (h *DNSHandler) isIPQuery(q dns.Question) int {
	if q.Qclass != dns.ClassINET {
		return notIPQuery
	}

	switch q.Qtype {
	case dns.TypeA:
		return _IP4Query
	case dns.TypeAAAA:
		return _IP6Query
	default:
		return notIPQuery
	}
}

// UnFqdn function
func UnFqdn(s string) string {
	if dns.IsFqdn(s) {
		return s[:len(s)-1]
	}
	return s
}
