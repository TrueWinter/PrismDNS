package main

import (
	"io"

	"codeberg.org/miekg/dns"
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