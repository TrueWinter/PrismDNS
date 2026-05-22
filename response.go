package main

import (
	"fmt"
	"io"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
)

type Response struct {}

func (r *Response) respond(m *dns.Msg, w dns.ResponseWriter) {
	m.RecursionAvailable = false
	m.Response = true
	m.Pack()
	io.Copy(w, m)
}

func (r *Response) updateMessageWithRcode(rcode uint16, m *dns.Msg) {
	m.Rcode = rcode
	m.Extra = []dns.RR{}
	m.Response = true
}

func (r *Response) respondWithRcode(rcode uint16, m *dns.Msg, w dns.ResponseWriter) {
	r.updateMessageWithRcode(rcode, m)
	r.respond(m, w)
}

func getVersionString() string {
	s := "PrismDNS"
	if Version != "" {
		s = fmt.Sprintf("%v %v", s, Version)
	}
	return s
}

func (r *Response) getChaosResponse(m *dns.Msg) *dns.TXT {
	queries := map[string]string{
		"version.bind": getVersionString(),
		"id.server": ServerId,
	}

	if t, ok := m.Question[0].(*dns.TXT); ok {
		response, exists := queries[NormalizeDomain(t.Hdr.Name)]
		if exists && response != "" {
			return &dns.TXT{
				Hdr: dns.Header{
					Name: t.Hdr.Name,
					Class: dns.ClassCHAOS,
					TTL: 3600,
				},
				TXT: rdata.TXT{
					Txt: []string{
						response,
					},
				},
			}
		}
	}

	return nil
}
