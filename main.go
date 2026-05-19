package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

type RouteFlags []string

func (i *RouteFlags) String() string {
	return fmt.Sprintf("%v", *i)
}

func (r *RouteFlags) Set(value string) error {
	*r = append(*r, value)
	return nil
}

var FALLBACK_ROUTE_DOMAIN = "_fallback_"

func main() {
	var routeFlags RouteFlags
	host := flag.String("host", "0.0.0.0", "Address that the DNS proxy runs on")
	port := flag.String("port", "53", "Port that the DNS proxy runs on")
	upstreamReadTimeout := flag.Int("upstream-read-timeout", 2, "Read timeout for upstream DNS requests")
	upstreamWriteTimeout := flag.Int("upstream-write-timeout", 8, "Write timeout for upstream DNS requests")
	clientReadTimeout := flag.Int("client-read-timeout", 2, "Read timeout for client DNS requests")
	clientIdleTimeout := flag.Int("client-idle-timeout", 8, "Idle timeout for client DNS requests")
	debug := flag.Bool("debug", false, "Enable debug logs")
	fallback := flag.String("fallback", "", "Fallback DNS server in the format <ip>[,<port>] used if no routes match. If unset, NXDOMAIN is returned.")
	flag.Var(&routeFlags, "route", "Route configuration in the format <domain>,<ip>[,<port>]. Can be used multiple times.")
	flag.Parse()

	if *fallback != "" {
		parts := strings.Split(*fallback, ",")

		var ip string
		// The port is parsed later
		var port string
		
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

	server := Server{
		Host: *host,
		Port: *port,
		Routes: routes,
		Debug: *debug,
		ClientReadTimeout:   *clientReadTimeout,
		ClientIdleTimeout:  *clientIdleTimeout,
	}

	// Create a channel to receive shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		server.Start()
	}()

	<-shutdown
	server.Stop()
}