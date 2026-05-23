package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
	if flags.HideVersion {
		Version = ""
	}

	router := &Router{
		Routes: make(map[string]*Route),
		UpstreamReadTimeout: flags.UpstreamReadTimeout,
		UpstreamWriteTimeout: flags.UpstreamWriteTimeout,
	}

	httpServer := &HttpServer{
		Host: flags.HttpHost,
		Port: flags.HttpPort,
		ApiKey: flags.HttpApiKey,
		Router: router,
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