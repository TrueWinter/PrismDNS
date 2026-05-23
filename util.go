package main

import (
	"fmt"
	"strconv"
	"strings"
)

func NormalizeDomain(domain string) string {
	if domain[len(domain)-1:] == "." {
		domain = domain[:len(domain)-1]
	}
	return strings.ToLower(domain)
}

func ParseHostPort(hostport string) (string, int) {
	if strings.Contains(hostport, ":") {
		parts := strings.SplitN(hostport, ":", 2)
		ip := parts[0]
		// If error, 0 is returned as port which can be checked by caller
		port, _ := strconv.Atoi(parts[1])
		return ip, port
	}
	return hostport, 53
}

func FormatAddr(host string, port int) string {
	return fmt.Sprintf("%v:%v", host, port)
}