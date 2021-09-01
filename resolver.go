package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// ResolvError type
type ResolvError struct {
	qname, net  string
	nameservers []string
}

// Error formats a ResolvError
func (e ResolvError) Error() string {
	errmsg := fmt.Sprintf("%s resolv failed on %s (%s)", e.qname, strings.Join(e.nameservers, "; "), e.net)
	return errmsg
}

// Resolver type
type Resolver struct {
	config *dns.ClientConfig
}

// Lookup will ask each nameserver in top-to-bottom fashion, starting a new request
// in every second, and return as early as possbile (have an answer).
// It returns an error if no request has succeeded.
func (r *Resolver) Lookup(net string, req *dns.Msg, timeout int, interval int, nameServers []string, DoH string) (message *dns.Msg, err error) {
	logger.Debugf("Lookup %s, timeout: %d, interval: %d, nameservers: %v, Using DoH: %v", net, timeout, interval, nameServers, DoH != "")

	//Is DoH enabled
	if DoH != "" {
		//First try and use DOH. Privacy First
		ans, err := r.DoHLookup(DoH, timeout, req)
		if err == nil {
			//No error so result is ok
			return ans, nil
		}

		//For some reason the DoH lookup failed so fall back to nameservers
		logger.Debugf("DoH Failed due to '%s' falling back to nameservers", err)

	}

	c := &dns.Client{
		Net:          net,
		ReadTimeout:  r.Timeout(timeout),
		WriteTimeout: r.Timeout(timeout),
	}

	qname := req.Question[0].Name

	res := make(chan *dns.Msg, 1)
	var wg sync.WaitGroup
	L := func(nameserver string) {
		defer wg.Done()
		r, _, err := c.Exchange(req, nameserver)
		if err != nil {
			logger.Errorf("%s socket error on %s", qname, nameserver)
			logger.Errorf("error:%s", err.Error())
			return
		}
		if r != nil && r.Rcode != dns.RcodeSuccess {
			logger.Warningf("%s failed to get an valid answer on %s", qname, nameserver)
			if r.Rcode == dns.RcodeServerFailure {
				return
			}
		} else {
			logger.Debugf("%s resolv on %s (%s)\n", UnFqdn(qname), nameserver, net)
		}
		select {
		case res <- r:
		default:
		}
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	// Start lookup on each nameserver top-down, in every second
	for _, nameServer := range nameServers {
		wg.Add(1)
		go L(nameServer)
		// but exit early, if we have an answer
		select {
		case r := <-res:
			return r, nil
		case <-ticker.C:
			continue
		}
	}

	// wait for all the namservers to finish
	wg.Wait()
	select {
	case r := <-res:
		return r, nil
	default:
		return nil, ResolvError{qname, net, nameServers}
	}
}

// Timeout returns the resolver timeout
func (r *Resolver) Timeout(timeout int) time.Duration {
	return time.Duration(timeout) * time.Second
}

//DoHLookup performs a DNS lookup over https
func (r *Resolver) DoHLookup(url string, timeout int, req *dns.Msg) (*dns.Msg, error) {
	qname := req.Question[0].Name

	//Turn message into wire format
	data, err := req.Pack()
	if err != nil {
		logger.Errorf("Failed to pack DNS message to wire format; %s", err)
		return nil, ResolvError{qname, "HTTPS", []string{url}}
	}

	//Make the request to the server
	client := http.Client{
		Timeout: r.Timeout(timeout),
	}

	reader := bytes.NewReader(data)
	resp, err := client.Post(url, "application/dns-message", reader)
	if err != nil {
		logger.Errorf("Request to DoH server failed; %s", err)
		return nil, ResolvError{qname, "HTTPS", []string{url}}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(resp.Body)

	//Check the request went ok
	if resp.StatusCode != http.StatusOK {
		return nil, ResolvError{qname, "HTTPS", []string{url}}
	}

	if resp.Header.Get("Content-Type") != "application/dns-message" {
		return nil, ResolvError{qname, "HTTPS", []string{url}}
	}

	//Unpack the message from the HTTPS response
	respPacket, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, ResolvError{qname, "HTTPS", []string{url}}
	}

	res := dns.Msg{}
	err = res.Unpack(respPacket)
	if err != nil {
		logger.Errorf("Failed to unpack message from response; %s", err)
		return nil, ResolvError{qname, "HTTPS", []string{url}}
	}

	//Finally return
	return &res, nil
}
