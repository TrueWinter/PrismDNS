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
	Router      *Router
	dns.Handler
	Response
}

type QueryResult struct {
	response *dns.Msg
	err      error
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
	prismdnsDNSRequestsTotal.Inc()

	questions := r.Question
	if len(questions) == 0 {
		if Debug {
			fmt.Printf("Received invalid request with ID %v: no questions\n", r.ID)
		}
		h.incrementServfail("invalid_request")
		prismdnsDNSResponsesTotal.WithLabelValues("SERVFAIL").Inc()
		h.respondWithRcode(dns.RcodeServerFailure, r, w)
		return
	}

	header := questions[0].Header()
	domain := NormalizeDomain(header.Name)

	if h.shouldRateLimit(domain, w.RemoteAddr()) && w.RemoteAddr().Network() == "udp" {
		r.Truncated = true
		prismdnsRateLimitHitsTotal.Inc()
		h.respond(r, w)
		return
	}

	if header.Class == dns.ClassCHAOS {
		chaosResponse := h.getChaosResponse(r)
		if chaosResponse != nil {
			r.Answer = append(r.Answer, chaosResponse)
			h.respond(r, w)
			prismdnsDNSResponsesTotal.WithLabelValues("NOERROR").Inc()
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
			response, err := route.Query(r)
			ch <- QueryResult{
				response,
				err,
			}
		}()

		result := <-ch

		if result.err != nil {
			prismdnsUpstreamResponsesTotal.WithLabelValues(route.Ip, "SERVFAIL").Inc()
			h.incrementServfail("upstream_failure")
			prismdnsDNSResponsesTotal.WithLabelValues("SERVFAIL").Inc()
			h.respondWithRcode(dns.RcodeServerFailure, r, w)
		} else {
			rcodeStr := rcodeToString(result.response.Rcode)
			prismdnsUpstreamResponsesTotal.WithLabelValues(route.Ip, rcodeStr).Inc()
			h.respond(result.response, w)
			prismdnsDNSResponsesTotal.WithLabelValues(rcodeStr).Inc()
		}
		return
	}

	if Debug {
		fmt.Printf("Returned NXDOMAIN for domain %v\n", domain)
	}
	prismdnsNXDOMAINTotal.Inc()
	prismdnsDNSResponsesTotal.WithLabelValues("NXDOMAIN").Inc()
	h.respondWithRcode(dns.RcodeNameError, r, w)
}

func (h *DnsHandler) incrementServfail(cause string) {
	prismdnsServfailTotal.WithLabelValues(cause).Inc()
}

func rcodeToString(rcode uint16) string {
	return dns.RcodeToString[rcode]
}
