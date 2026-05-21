package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type RouteFlags []string

func (r *RouteFlags) String() string {
	return fmt.Sprintf("%v", *r)
}

func (r *RouteFlags) Set(value string) error {
	*r = append(*r, value)
	return nil
}

type DomainRateLimitFlags []string

func (d *DomainRateLimitFlags) String() string {
	return fmt.Sprintf("%v", *d)
}

func (d *DomainRateLimitFlags) Set(value string) error {
	*d = append(*d, value)
	return nil
}

type Flags struct {
	Host string
	Port int
	Routes []Route
	Fallback string
	RateLimit int
	RateLimitWindow int
	DomainRateLimits map[string]int
	LogRateLimits bool
	UpstreamReadTimeout int
	UpstreamWriteTimeout int
	ClientReadTimeout int
	ClientIdleTimeout int
	Debug bool
	HideVersion bool
	ServerId string
}

func ParseFlags(softwareVersion string) Flags {
	var routeFlags RouteFlags
	var domainRateLimitFlags DomainRateLimitFlags
	host := flag.String("host", "0.0.0.0", "Address that the DNS proxy runs on")
	port := flag.Int("port", 53, "Port that the DNS proxy runs on")
	upstreamReadTimeout := flag.Int("upstream-read-timeout", 2, "Read timeout for upstream DNS requests")
	upstreamWriteTimeout := flag.Int("upstream-write-timeout", 8, "Write timeout for upstream DNS requests")
	clientReadTimeout := flag.Int("client-read-timeout", 2, "Read timeout for client DNS requests")
	clientIdleTimeout := flag.Int("client-idle-timeout", 8, "Idle timeout for client DNS requests")
	debug := flag.Bool("debug", false, "Enable debug logs")
	fallback := flag.String("fallback", "", "Fallback DNS server in the format <ip>[,<port>] used if no routes match. If unset, NXDOMAIN is returned.")
	// 15000 per minute equals 250 queries per second
	rateLimit := flag.Int("rate-limit", 15000, "Maximum requests per window for all clients (set to 0 to disable)")
	rateLimitWindow := flag.Int("rate-limit-window", 60, "Time window in seconds for ratelimiting")
	logRateLimits := flag.Bool("log-rate-limits", true, "Log rate limits at most once every 60 seconds")
	version := flag.Bool("version", false, "Get dns-proxy version")
	hideVersion := flag.Bool("hide-version", false, "Hide the version number in DNS responses")
	serverId := flag.String("server-id", "", "Server ID used in DNS responses")
	flag.Var(&routeFlags, "route", "Route configuration in the format <domain>,<ip>[,<port>]. Can be used multiple times.")
	flag.Var(&domainRateLimitFlags, "domain-rate-limit", "Ratelimit overrides for specific domains in the format <domain>,<rate-limit>")
	flag.Parse()

	if *version {
		fmt.Printf("Running dns-proxy version %v\n", softwareVersion)
		os.Exit(0)
	}

	if *fallback != "" {
		parts := strings.Split(*fallback, ",")

		var ip string
		// The port is parsed later
		port := "53"
		
		switch len(parts) {
			case 1:
				ip = parts[0]
			case 2:
				ip = parts[0]
				port = parts[1]
			default:
				fmt.Fprintf(os.Stderr, "Invalid fallback format: %s (expected 'ip' or 'ip,port')\n", *fallback)
				flag.Usage()
				os.Exit(1)
		}
		
		routeFlags = append(routeFlags, fmt.Sprintf("%v,%v,%v", FALLBACK_ROUTE_DOMAIN, ip, port))
	}

	routes := []Route{}

	for _, route := range routeFlags {
		parts := strings.Split(route, ",")
		if len(parts) != 2 && len(parts) != 3 {
			fmt.Fprintf(os.Stderr, "Invalid route format: %s (expected 'domain,ip' or 'domain,ip,port')\n", route)
			flag.Usage()
			os.Exit(1)
		}

		domain := strings.ToLower(parts[0])
		ip := parts[1]
		port := 53

		if domain == "" || ip == "" {
			fmt.Fprintln(os.Stderr, "Domain and IP cannot be blank")
			flag.Usage()
			os.Exit(1)
		}

		if len(parts) == 3 {
			p, err := strconv.Atoi(parts[2])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Invalid port number")
				flag.Usage()
				os.Exit(1)
			}
			port = p
		}

		if domain == FALLBACK_ROUTE_DOMAIN {
			domain = ""
		}

		routes = append(routes, Route{
			Domain: domain,
			Ip: ip,
			Port: port,
			UpstreamReadTimeout: *upstreamReadTimeout,
			UpstreamWriteTimeout: *upstreamWriteTimeout,
			Debug: *debug,
		})
	}

	if len(routes) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No routes configured. Please use -route flag with at least one route configuration.")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("Configured routes:")
	for _, route := range routes {
		domain := route.Domain
		if domain == "" {
			domain = FALLBACK_ROUTE_DOMAIN
		}
		fmt.Printf("  - %v -> %v:%v\n", domain, route.Ip, route.Port)
	}

	domainRateLimits := make(map[string]int)

	for _, domainRateLimit := range domainRateLimitFlags {
		parts := strings.Split(domainRateLimit, ",")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid domain rate limit format '%s', expected 'domain,rate'\n", domainRateLimit)
			os.Exit(1)
		}

		domain := parts[0]
		rate, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid rate limit value '%s' for domain '%s', must be an integer\n", parts[1], domain)
			os.Exit(1)
		}
		domainRateLimits[domain] = rate
	}

	return Flags{
		Host: *host,
		Port: *port,
		Routes: routes,
		Fallback: *fallback,
		UpstreamReadTimeout: *upstreamReadTimeout,
		UpstreamWriteTimeout: *upstreamWriteTimeout,
		ClientReadTimeout: *clientReadTimeout,
		ClientIdleTimeout: *clientIdleTimeout,
		RateLimit: *rateLimit,
		RateLimitWindow: *rateLimitWindow,
		DomainRateLimits: domainRateLimits,
		LogRateLimits: *logRateLimits,
		Debug: *debug,
		HideVersion: *hideVersion,
		ServerId: *serverId,
	}
}
