package main

import "strings"

func NormalizeDomain(domain string) string {
	if domain[len(domain)-1:] == "." {
		domain = domain[:len(domain)-1]
	}
	return strings.ToLower(domain)
}
