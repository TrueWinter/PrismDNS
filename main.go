package main

import (
	"os"
	"os/signal"
	"syscall"
)

// Set by build script
var Version string
var ServerId string
var FALLBACK_ROUTE_DOMAIN = "_fallback_"

func main() {
	flags := ParseFlags(Version)

	ServerId = flags.ServerId
	if flags.HideVersion {
		Version = ""
	}

	server := Server{
		Host: flags.Host,
		Port: flags.Port,
		Routes: flags.Routes,
		Debug: flags.Debug,
		ClientReadTimeout: flags.ClientReadTimeout,
		ClientIdleTimeout: flags.ClientIdleTimeout,
		RateLimiter: NewRateLimiter(
			flags.RateLimit,
			flags.RateLimitWindow,
			flags.DomainRateLimits,
			flags.LogRateLimits,
		),
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