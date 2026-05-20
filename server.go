package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"codeberg.org/miekg/dns"
)

type Server struct {
	Host string
	Port int
	Routes []Route
	Debug bool
	UpstreamReadTimeout int
	UpstreamWriteTimeout int
	ClientReadTimeout int
	ClientIdleTimeout int
	RateLimiter *RateLimiter
}

var udpServer *dns.Server
var tcpServer *dns.Server

func (s *Server) resolveAddr() string {
	return s.Host + ":" + strconv.Itoa(s.Port);
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
			Routes: s.Routes,
			Debug: s.Debug,
			RateLimiter: s.RateLimiter,
		},
	}
}

func (s *Server) Start() {
	udpServer = s.newServer("udp")
	tcpServer = s.newServer("tcp")

	go s.serve(udpServer)
	go s.serve(tcpServer)
	fmt.Printf("Listening on %v\n", s.resolveAddr())
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
