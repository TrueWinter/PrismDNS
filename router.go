package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"codeberg.org/miekg/dns"
)

type Route struct {
	Ip string
	Port int
	UpstreamReadTimeout int
	UpstreamWriteTimeout int
}

type Router struct {
	Routes map[string]*Route
	UpstreamReadTimeout int
	UpstreamWriteTimeout int
}

func (r *Route) resolveAddr() string {
	return FormatAddr(r.Ip, r.Port);
}

func (r *Router) getRouteStrict(domain string) *Route {
	route, exists := r.Routes[domain]
	if !exists {
		return nil
	}
	return route
}

func (r *Router) getRoute(domain string) *Route {
	var bestMatch string
	fallback, fallbackExists := r.Routes[FALLBACK_ROUTE_DOMAIN]

	if r := r.getRouteStrict(domain); r != nil {
		return r
	}

	for routeDomain := range r.Routes {
		if strings.HasSuffix(domain, routeDomain) {
			if bestMatch == "" || len(routeDomain) > len(bestMatch) {
				bestMatch = routeDomain
			}
		}
	}

	if bestMatch == "" && fallbackExists {
		return fallback
	}
	route, exists := r.Routes[bestMatch]
	if !exists {
		return nil
	}
	return route
}

func (r *Router) AddRoute(domain string, ip string, port int) error {
	existingRoute := r.getRouteStrict(domain)
	if existingRoute != nil {
		fmt.Println(existingRoute)
		return fmt.Errorf("Route for domain %v already exists", domain)
	}
	r.Routes[domain] = &Route{
		Ip: ip,
		Port: port,
		UpstreamReadTimeout: r.UpstreamReadTimeout,
		UpstreamWriteTimeout: r.UpstreamWriteTimeout,
	}
	return nil
}

func (r *Router) ModifyRoute(domain string, ip string, port int) error {
	route := r.getRouteStrict(domain)
	if route == nil {
		return errors.New("Route does not exist")
	}
	route.Ip = ip
	route.Port = port
	return nil
}

func (r *Router) DeleteRoute(domain string) error {
	route := r.getRouteStrict(domain)
	if route == nil {
		return errors.New("Route does not exist")
	}
	delete(r.Routes, domain)
	return nil
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
		if Debug {
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
