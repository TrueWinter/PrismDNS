package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"codeberg.org/miekg/dns"
)

type HealthCheckStatus struct {
	LastCheck time.Time
	LastErr string
	IsHealthy bool
}

type HealthCheckManager struct {
	router   *Router
	status   map[string]*HealthCheckStatus
	statusMu sync.RWMutex
}

type RouteInfo struct {
	Domain string
	Route  *Route
}

func NewHealthCheckManager(router *Router) *HealthCheckManager {
	return &HealthCheckManager{
		router:   router,
		status:   make(map[string]*HealthCheckStatus),
	}
}

func (h *HealthCheckManager) GetStatus(domain string) *HealthCheckStatus {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()
	if status, ok := h.status[domain]; ok {
		return status
	}
	return &HealthCheckStatus{
		LastCheck: time.Time{},
		IsHealthy: true,
	}
}

func (h *HealthCheckManager) GetAllStatuses() map[string]*HealthCheckStatus {
	return h.status
}

func (h *HealthCheckManager) DeleteStatus(domain string) {
	h.statusMu.Lock()
	defer h.statusMu.Unlock()
	delete(h.status, domain)
}

func (h *HealthCheckManager) PerformHealthCheck(info RouteInfo) error {
	if Debug {
		fmt.Printf(
			"Performing health check on route %v (%v:%v)\n",
			info.Domain, info.Route.Ip, info.Route.Port,
		)
	}

	m := &dns.Msg{
		Question: []dns.RR{
			&dns.A{
				Hdr: dns.Header{
					Name: fmt.Sprintf("%v.", info.Domain),
					Class: dns.ClassINET,
					TTL: 300,
				},
			},
		},
	}

	// Handle edge case if route is deleted before healthcheck finishes
	if info.Route == nil {
		return nil
	}
	_, err := info.Route.Query(m)

	var status HealthCheckStatus
	if err != nil {
		status = HealthCheckStatus{
			LastErr:  err.Error(),
			IsHealthy: false,
		}
		fmt.Fprintf(
			os.Stderr, "Healthcheck failed for %v (%v:%v): %v",
			info.Domain, info.Route, info.Route.Ip, err.Error(),
		)
	} else {
		status = HealthCheckStatus{
			IsHealthy: true,
			LastErr:   "",
		}
	}

	// Handle edge case if route is deleted before healthcheck finishes
	if info.Route == nil {
		return nil
	}

	h.statusMu.Lock()
	defer h.statusMu.Unlock()

	status.LastCheck = time.Now()
	h.status[info.Domain] = &status

	if Debug {
		fmt.Printf(
			"Health check status for route %v (%v:%v): %v\n",
			info.Domain, info.Route.Ip, info.Route.Port, status,
		)
	}

	return nil
}

func (h *HealthCheckManager) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			h.CheckAllRoutes()
		}
	}()
}

func (h *HealthCheckManager) getRoutes() []RouteInfo {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()
	routes := make([]RouteInfo, 0, len(h.router.Routes))
	for domain, route := range h.router.Routes {
		routes = append(routes, RouteInfo{
			Domain: domain,
			Route: route,
		})
	}
	return routes
}

func (h *HealthCheckManager) CheckAllRoutes() {
	if Debug {
		fmt.Println("Performing health checks")
	}
	routes := h.getRoutes()
	for _, rc := range routes {
		go func(info RouteInfo) {
			jitter := time.Duration(rand.Intn(2000)) * time.Millisecond
			time.Sleep(jitter)
			h.PerformHealthCheck(info)
		}(rc)
	}
}
