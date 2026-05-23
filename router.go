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
	UpstreamReadTimeout int
	UpstreamWriteTimeout int
}

type Router struct {
	Routes map[string]*Route
	RouteMu sync.RWMutex
	UpstreamReadTimeout int
	UpstreamWriteTimeout int
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
		UpstreamReadTimeout: r.UpstreamReadTimeout,
		UpstreamWriteTimeout: r.UpstreamWriteTimeout,
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
				fmt.Printf("UDP connection to %v failed, retrying over TCP: %v\n", r.resolveAddr(), err)
			} else {
				fmt.Printf("UDP response to %v truncated, retrying over TCP: %v\n", r.resolveAddr(), err)
			}
		}

		tcpUpstream := dns.NewClient()
		upstream.ReadTimeout = time.Duration(r.UpstreamReadTimeout) * time.Second
		upstream.WriteTimeout = time.Duration(r.UpstreamWriteTimeout) * time.Second

		response, _, err = tcpUpstream.Exchange(context.TODO(), m, "tcp", r.resolveAddr())
	}

	return response, err
}
