package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"codeberg.org/miekg/dns"
)

type Route struct {
	Domain string
	Ip string
	Port int
	UpstreamReadTimeout int
	UpstreamWriteTimeout int
	Debug bool
	Response
}

func (r *Route) resolveAddr() string {
	return r.Ip + ":" + strconv.Itoa(r.Port);
}

func (r *Route) Query(m *dns.Msg) (*dns.Msg, error) {
	m.RecursionDesired = false
	m.Authoritative = false
	m.Rcode = dns.RcodeSuccess

	upstream := dns.NewClient()
	upstream.ReadTimeout = time.Duration(r.UpstreamReadTimeout) * time.Second
	upstream.WriteTimeout = time.Duration(r.UpstreamWriteTimeout) * time.Second
	
	response, _, err := upstream.Exchange(context.TODO(), m, "udp", r.resolveAddr())

	if response == nil || response.Truncated {
		if r.Debug {
			if response == nil {
				fmt.Printf("UDP connection to %v failed, retrying over TCP\n", r.resolveAddr())
			} else {
				fmt.Printf("UDP response to %v truncated, retrying over TCP\n", r.resolveAddr())
			}
		}

		tcpUpstream := dns.NewClient()
		upstream.ReadTimeout = time.Duration(r.UpstreamReadTimeout) * time.Second
		upstream.WriteTimeout = time.Duration(r.UpstreamWriteTimeout) * time.Second

		response, _, err = tcpUpstream.Exchange(context.TODO(), m, "tcp", r.resolveAddr())
	}

	return response, err
}
