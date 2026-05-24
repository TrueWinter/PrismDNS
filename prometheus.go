package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	prismdnsDNSRequestsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prismdns_dns_requests_total",
			Help: "Total DNS requests received by the proxy",
		},
	)

	prismdnsDNSResponsesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prismdns_dns_responses_total",
			Help: "Total DNS responses sent by the proxy",
		},
		[]string{"rcode"},
	)

	prismdnsUpstreamQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prismdns_upstream_queries_total",
			Help: "Total upstream DNS queries forwarded by the proxy",
		},
		[]string{"upstream", "protocol"},
	)

	prismdnsUpstreamResponsesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prismdns_upstream_responses_total",
			Help: "Total upstream DNS responses received by the proxy",
		},
		[]string{"upstream", "rcode"},
	)

	prismdnsNXDOMAINTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prismdns_nxdomain_total",
			Help: "Total DNS responses with RCODE NameError (NXDOMAIN)",
		},
	)

	prismdnsServfailTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prismdns_servfail_total",
			Help: "Total DNS responses with RCODE Server Failure",
		},
		[]string{"cause"},
	)

	prismdnsRateLimitHitsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prismdns_ratelimit_hits_total",
			Help: "Total DNS requests blocked due to rate limiting",
		},
	)

	prismdnsUpstreamHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prismdns_upstream_health",
			Help: "Health status of upstream servers",
		},
		[]string{"upstream"},
	)

	prismdnsUpstreamLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prismdns_upstream_latency",
			Help: "Latency of upstream servers",
		},
		[]string{"upstream"},
	)
)

func RegisterMetrics() {
	prometheus.MustRegister(
		prismdnsDNSRequestsTotal,
		prismdnsDNSResponsesTotal,
		prismdnsUpstreamQueriesTotal,
		prismdnsUpstreamResponsesTotal,
		prismdnsNXDOMAINTotal,
		prismdnsServfailTotal,
		prismdnsRateLimitHitsTotal,
		prismdnsUpstreamHealth,
		prismdnsUpstreamLatency,
	)
}

func GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}
