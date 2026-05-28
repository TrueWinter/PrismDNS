package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
)

// Set by build script
var Version string
var ServerId string
var Debug bool
var FALLBACK_ROUTE_DOMAIN = "_fallback_"

func main() {
	flags := ParseFlags()

	ServerId = flags.ServerId
	Debug = flags.Debug

	fmt.Printf("Running PrismDNS version %v\nServer ID: %v\nDebug mode: %v\n", Version, ServerId, Debug)

	if flags.HideVersion {
		Version = ""
	}

	if len(flags.ParsedRoutes) == 0 {
		fmt.Printf("No routes configured. Please use -route flag with at least one route configuration or add routes through the HTTP API.")
	}

	fmt.Println("Configured routes:")
	for _, route := range flags.ParsedRoutes {
		fmt.Printf("  - %v -> %v:%v\n", route.Domain, route.Ip, route.Port)
	}

	router := &Router{
		Routes: make(map[string]*Route),
		Timeout: flags.UpstreamTimeout,
	}

	metricsRegistry := prometheus.NewRegistry()
	RegisterMetrics()

	httpServer := &HttpServer{
		Host: flags.HttpHost,
		Port: flags.HttpPort,
		ApiKey: flags.HttpApiKey,
		Router: router,
		MetricsRegistry: metricsRegistry,
	}

	server := Server{
		Host: flags.Host,
		Port: flags.Port,
		HttpServer: httpServer,
		Router: router,
		ClientReadTimeout: flags.ClientReadTimeout,
		ClientIdleTimeout: flags.ClientIdleTimeout,
		RateLimiter: NewRateLimiter(
			flags.RateLimit,
			flags.RateLimitWindow,
			flags.DomainRateLimits,
			flags.LogRateLimits,
		),
	}

	for _, route := range flags.ParsedRoutes {
		err := router.AddRoute(route.Domain, route.Ip, route.Port)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occurred while adding route %v: %v\n", route.Domain, err.Error())
		}
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
