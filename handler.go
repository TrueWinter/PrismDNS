package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"codeberg.org/miekg/dns"
)

type DnsHandler struct {
	RateLimiter *RateLimiter
	Router *Router
	dns.Handler
	Response
}

type QueryResult struct {
	response *dns.Msg
	err error
}

func (h *DnsHandler) shouldRateLimit(domain string, addr net.Addr) bool {
	ip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse remote address %v, defaulting to rate limit\n", addr.String())
		return true
	}

	return h.RateLimiter.ShouldRateLimit(domain, ip)
}

func (h *DnsHandler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) {
	r.Unpack()
	m := r.Copy()

	questions := r.Question
	if len(questions) == 0 {
		if Debug {
			fmt.Printf("Received invalid request with ID %v: no questions\n", r.ID)
		}
		h.respondWithRcode(dns.RcodeServerFailure, m, w)
		return
	}

	header := questions[0].Header()
	domain := NormalizeDomain(header.Name)

	if h.shouldRateLimit(domain, w.RemoteAddr()) && w.RemoteAddr().Network() == "udp" {
		r.Truncated = true
		h.respond(r, w)
		return
	}

	if header.Class == dns.ClassCHAOS {
		chaosResponse := h.getChaosResponse(r)
		if chaosResponse != nil {
			m.Answer = append(r.Answer, chaosResponse)
			h.respond(m, w)
			return
		}
	}

	route := h.Router.getRoute(domain)
	if route != nil {
		if Debug {
			fmt.Printf("Making DNS request to %v:%v for domain %v\n", route.Ip, route.Port, domain)
		}

		ch := make(chan QueryResult, 1)
		go func() {
			response, err := route.Query(m)
			ch <- QueryResult{
				response,
				err,
			}
		}()

		result := <- ch

		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query upstream for %s: %v\n", domain, result.err)
			h.respondWithRcode(dns.RcodeServerFailure, m, w)
			return
		}

		m.Answer = result.response.Answer
		m.Ns = result.response.Ns
		m.Pseudo = result.response.Pseudo
		h.respond(m, w)
		return
	}

	if Debug {
		fmt.Printf("Returned NXDOMAIN for domain %v\n", domain)
	}
	
	h.respondWithRcode(dns.RcodeNameError, m, w)
}