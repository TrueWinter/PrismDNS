package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"codeberg.org/miekg/dns"
)

type Route struct {
	Ip string
	Port int
	Timeout int
}

type Router struct {
	Routes map[string]*Route
	RouteMu sync.RWMutex
	Timeout int
	HealthCheck *HealthCheckManager
}

func (r *Route) resolveAddr() string {
	return FormatAddr(r.Ip, r.Port)
}

func (r *Router) getRouteStrict(domain string) *Route {
	// If the mutex is already locked, don't try to relock it
	if r.RouteMu.TryRLock() {
		defer r.RouteMu.RUnlock()
	}
	route, exists := r.Routes[domain]
	if !exists {
		return nil
	}
	return route
}

func (r *Router) getRoute(domain string) *Route {
	r.RouteMu.RLock()
	defer r.RouteMu.RUnlock()
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
	r.RouteMu.Lock()
	defer r.RouteMu.Unlock()
	existingRoute := r.getRouteStrict(domain)
	if existingRoute != nil {
		return fmt.Errorf("Route for domain %v already exists", domain)
	}
	r.Routes[domain] = &Route{
		Ip: ip,
		Port: port,
		Timeout:  r.Timeout,
	}
	return nil
}

func (r *Router) ModifyRoute(domain string, ip string, port int) error {
	r.RouteMu.Lock()
	defer r.RouteMu.Unlock()
	route := r.getRouteStrict(domain)
	if route == nil {
		return errors.New("Route does not exist")
	}
	route.Ip = ip
	route.Port = port
	return nil
}

func (r *Router) DeleteRoute(domain string) error {
	r.RouteMu.Lock()
	defer r.RouteMu.Unlock()
	route := r.getRouteStrict(domain)
	if route == nil {
		return errors.New("Route does not exist")
	}

	if r.HealthCheck != nil {
		r.HealthCheck.DeleteStatus(domain)
	}

	delete(r.Routes, domain)
	return nil
}

func (r *Route) directQuery(m *dns.Msg) (*dns.Msg, string, error) {
	m.RecursionDesired = false
	m.Authoritative = false
	m.Rcode = dns.RcodeSuccess

	timeout := time.Duration(r.Timeout) * time.Second

	protocol := "udp"
	upstream := dns.NewClient()
	upstreamCtx, upstreamCancel := context.WithTimeout(context.Background(), timeout)
	defer upstreamCancel()
	response, _, err := upstream.Exchange(upstreamCtx, m, "udp", r.resolveAddr())

	if response == nil || response.Truncated {
		if Debug {
			if response == nil {
				fmt.Printf("UDP connection to %v failed, retrying over TCP: %v\n", r.resolveAddr(), err)
			} else {
				fmt.Printf("UDP response to %v truncated, retrying over TCP: %v\n", r.resolveAddr(), err)
			}
		}

		protocol = "tcp"
		tcpUpstream := dns.NewClient()
		tcpUpstreamCtx, tcpUpstreamCancel := context.WithTimeout(context.Background(), timeout)
		defer tcpUpstreamCancel()
		response, _, err = tcpUpstream.Exchange(tcpUpstreamCtx, m, "tcp", r.resolveAddr())
	}

	return response, protocol, err
}

func (r *Route) Query(m *dns.Msg) (*dns.Msg, error) {
	msg, protocol, err := r.directQuery(m)
	prismdnsUpstreamQueriesTotal.WithLabelValues(FormatAddr(r.Ip, r.Port), "udp").Inc()
	if protocol == "tcp" {
		prismdnsUpstreamQueriesTotal.WithLabelValues(FormatAddr(r.Ip, r.Port), "tcp").Inc()
	}
	return msg, err
}
