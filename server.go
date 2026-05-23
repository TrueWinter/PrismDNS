package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"codeberg.org/miekg/dns"
)

type Server struct {
	Host string
	Port int
	Router *Router
	Debug bool
	UpstreamReadTimeout int
	UpstreamWriteTimeout int
	ClientReadTimeout int
	ClientIdleTimeout int
	RateLimiter *RateLimiter
	HttpServer *HttpServer
}

var udpServer *dns.Server
var tcpServer *dns.Server

func (s *Server) resolveAddr() string {
	return FormatAddr(s.Host, s.Port)
}

func (s *Server) serve(server *dns.Server) {
	err := server.ListenAndServe()
	if err != nil {
		fmt.Fprintln(os.Stderr,"An error occurred starting DNS server", err.Error())
	}
}

func (s *Server) newServer(net string) *dns.Server {
	return &dns.Server{
		Addr: s.resolveAddr(),
		Net: net,
		ReadTimeout: time.Duration(s.ClientReadTimeout) * time.Second,
		IdleTimeout: time.Duration(s.ClientIdleTimeout) * time.Second,
		Handler: &DnsHandler{
			Router: s.Router,
			RateLimiter: s.RateLimiter,
		},
	}
}

func (s *Server) Start() {
	udpServer = s.newServer("udp")
	tcpServer = s.newServer("tcp")

	addr := s.resolveAddr()
	fmt.Printf("Starting DNS server on %v\n", addr)
	go s.serve(udpServer)
	go s.serve(tcpServer)
	fmt.Printf("DNS server listening on %v\n", addr)

	httpAddr := FormatAddr(s.HttpServer.Host, s.HttpServer.Port)
	fmt.Printf("Starting HTTP server on %v\n", httpAddr)
	go s.HttpServer.Start()
	fmt.Printf("HTTP server listening on %v\n", httpAddr)
}

func (s *Server) Stop() {
	fmt.Println("Stopping server")

	if udpServer != nil {
		udpServer.Shutdown(context.TODO())
	}

	if tcpServer != nil {
		tcpServer.Shutdown(context.TODO())
	}
}
