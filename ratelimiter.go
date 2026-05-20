package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type RateLimitClient struct {
	Requests int64
	WindowStart time.Time
	LastLog time.Time
}

type RateLimiter struct {
	RateLimit int
	RateLimitWindow int
	DomainRateLimits map[string]int
	Clients map[string]*RateLimitClient
	ClientsMu sync.Mutex
	LogRateLimits bool
}

func (rl *RateLimiter) getLimitForDomain(domain string) int {
	domainRateLimit, exists := rl.DomainRateLimits[domain]
	if exists {
		return domainRateLimit
	}
	return rl.RateLimit
}

func (rl *RateLimiter) normalizeIp(ip string) string {
	parsedIp := net.ParseIP(ip)
	if parsedIp == nil {
		return ""
	}

	v4 := parsedIp.To4()
	if v4 != nil {
		return v4.String()
	}

	return parsedIp.Mask(net.CIDRMask(64, 128)).String()
}

func NewRateLimiter(
	RateLimit int,
	RateLimitWindow int,
	DomainRateLimits map[string]int,
	LogRateLimits bool,
) *RateLimiter {
	rl := &RateLimiter{
		RateLimit: RateLimit,
		RateLimitWindow: RateLimitWindow,
		DomainRateLimits: DomainRateLimits,
		Clients: make(map[string]*RateLimitClient),
		LogRateLimits: LogRateLimits,
	}

	go func() {
		cleanupTicker := time.NewTicker(5 * time.Minute)
		defer cleanupTicker.Stop()

		for range cleanupTicker.C {
			rl.CleanupExpired()
		}
	}()

	return rl
}

func (rl *RateLimiter) ShouldRateLimit(domain string, clientIP string) bool {
	ip := rl.normalizeIp(clientIP)
	if ip == "" {
		fmt.Fprintf(os.Stderr, "Failed to parse IP %v, defaulting to rate limit\n", ip)
		return true
	}

	rl.ClientsMu.Lock()
	defer rl.ClientsMu.Unlock()

	if rl.RateLimit == 0 && len(rl.DomainRateLimits) == 0 {
		return false
	}

	client, exists := rl.Clients[ip]
	if !exists {
		client = &RateLimitClient{
			Requests: 0,
			WindowStart: time.Now(),
			// The day that DNS proxy was created as a proof-of-concept
			LastLog: time.Date(2026, 05, 18, 0, 0, 0, 0, time.UTC),
		}
		rl.Clients[ip] = client
	}

	if time.Since(client.WindowStart) > time.Duration(rl.RateLimitWindow) * time.Second {
		client.Requests = 0
		client.WindowStart = time.Now()
	}

	domainLimit := rl.getLimitForDomain(domain)
	if domainLimit == 0 {
		return false
	}

	if client.Requests >= int64(domainLimit) {
		if rl.LogRateLimits && time.Since(client.LastLog) > 60 * time.Second {
			fmt.Printf(
				"Client request ratelimited: %v (normalized %v) requested %v times in %v seconds, limit %v\n",
				clientIP, ip, client.Requests, rl.RateLimitWindow, domainLimit,
			)
			client.LastLog = time.Now()
		}
		return true
	}

	client.Requests++
	return false
}

func (rl *RateLimiter) CleanupExpired() {
	rl.ClientsMu.Lock()
	defer rl.ClientsMu.Unlock()

	for ip, client := range rl.Clients {
		if time.Since(client.WindowStart) > time.Duration(rl.RateLimitWindow) * time.Second {
			delete(rl.Clients, ip)
		}
	}
}
