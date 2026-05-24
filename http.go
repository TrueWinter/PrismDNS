package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type HttpServer struct {
	Host string
	Port int
	ApiKey string
	Router *Router
	HealthCheck *HealthCheckManager
}

type HttpError struct {
	Error string
}

type HttpRouteObject struct {
	Domain string
	Ip string
	Port int
}

type HttpModifyRouteObject struct {
	Ip string
	Port int
}

type HttpHealthCheckObject struct {
	Ok bool
}

type HttpSuccessObject struct {
	Success bool
}

func convertRouteToHttpRouteObject(domain string, route Route) HttpRouteObject {
	return HttpRouteObject{
		Domain: domain,
		Ip: route.Ip,
		Port: route.Port,
	}
}

type MiddlewareRouter struct {
	apiKey string
	next http.Handler
}

func (m *MiddlewareRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	serverApiKey := m.apiKey
	userApiKey := r.Header.Get("X-Api-Key")
	if serverApiKey != "" {
		serverApiKeyHash := sha256.Sum256([]byte(serverApiKey))
		userApiKeyHash := sha256.Sum256([]byte(userApiKey))
		if userApiKey == "" || subtle.ConstantTimeCompare(serverApiKeyHash[:], userApiKeyHash[:]) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(HttpError{
				Error: "Unautorized",
			})
			return
		}
	}

	m.next.ServeHTTP(w, r)
}

var server *http.Server

func (h *HttpServer) Start() {
	router := http.NewServeMux()
	server = &http.Server{
		Addr: FormatAddr(h.Host, h.Port),
		Handler: &MiddlewareRouter{
			apiKey: h.ApiKey,
			next: router,
		},
	}

	router.HandleFunc("GET /routes", func(w http.ResponseWriter, r *http.Request) {
		resp := []HttpRouteObject{}
		for domain, route := range h.Router.Routes {
			resp = append(resp, convertRouteToHttpRouteObject(domain, *route))
		}
		json.NewEncoder(w).Encode(resp)
	})

	router.HandleFunc("POST /routes", func(w http.ResponseWriter, r *http.Request) {
		var route HttpRouteObject
		json.NewDecoder(r.Body).Decode(&route)

		if route.Domain == "" {
			writeError("Domain is required", http.StatusBadRequest, w)
			return
		}

		if route.Ip == "" {
			writeError("IP is required", http.StatusBadRequest, w)
			return
		}

		if route.Port == 0 {
			writeError("Port is required", http.StatusBadRequest, w)
			return
		}

		err := h.Router.AddRoute(route.Domain, route.Ip, route.Port)
		if err != nil {
			writeError(err.Error(), 400, w)
			return
		}
		fmt.Printf("Route %v -> %v:%v added through API\n", route.Domain, route.Ip, route.Port)
		writeSuccess(w)
	})

	router.HandleFunc("GET /routes/{route}", func(w http.ResponseWriter, r *http.Request) {
		routeParam := r.PathValue("route")
		route := h.Router.getRouteStrict(routeParam)

		if route == nil {
			writeError("No matching route found", http.StatusNotFound, w)
			return
		}

		json.NewEncoder(w).Encode(convertRouteToHttpRouteObject(routeParam, *route))
	})

	router.HandleFunc("PUT /routes/{route}", func(w http.ResponseWriter, r *http.Request) {
		routeParam := r.PathValue("route")
		existingRoute := h.Router.getRouteStrict(routeParam)
		if existingRoute == nil {
			writeError("Route does not exist", 404, w)
			return
		}

		var route HttpModifyRouteObject
		json.NewDecoder(r.Body).Decode(&route)

		if route.Ip == "" {
			writeError("IP is required", http.StatusBadRequest, w)
			return
		}

		if route.Port == 0 {
			writeError("Port is required", http.StatusBadRequest, w)
			return
		}

		err := h.Router.ModifyRoute(routeParam, route.Ip, route.Port)
		if err != nil {
			writeError(err.Error(), 400, w)
			return
		}
		fmt.Printf("Route %v modified to %v:%v through API\n", routeParam, route.Ip, route.Port)
		writeSuccess(w)
	})

	router.HandleFunc("DELETE /routes/{route}", func(w http.ResponseWriter, r *http.Request) {
		routeParam := r.PathValue("route")
		existingRoute := h.Router.getRouteStrict(routeParam)
		if existingRoute == nil {
			writeError("Route does not exist", 404, w)
			return
		}

		if h.HealthCheck != nil {
			h.HealthCheck.DeleteStatus(routeParam)
		}

		err := h.Router.DeleteRoute(routeParam)
		if err != nil {
			writeError(err.Error(), 400, w)
			return
		}
		fmt.Printf("Route %v deleted through API\n", routeParam)
		writeSuccess(w)
	})

	router.HandleFunc("GET /healthcheck", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w)
	})

	router.HandleFunc("GET /healthcheck/routes", func(w http.ResponseWriter, r *http.Request) {
		statusMap := h.HealthCheck.GetAllStatuses()
		routes := []HealthCheckRouteResult{}
		for domain, status := range statusMap {
			result := formatHealthCheckStatus(domain, status)
			routes = append(routes, result)
		}
		json.NewEncoder(w).Encode(routes)
	})

	router.HandleFunc("GET /healthcheck/routes/{route}", func(w http.ResponseWriter, r *http.Request) {
		routeParam := r.PathValue("route")
		existingRoute := h.Router.getRouteStrict(routeParam)
		if existingRoute == nil {
			writeError("Route does not exist", 404, w)
			return
		}
		status := h.HealthCheck.GetStatus(routeParam)
		result := formatHealthCheckStatus(routeParam, status)
		json.NewEncoder(w).Encode(result)
	})

	err := server.ListenAndServe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start HTTP server: %v\n", err)
	}
}

type HealthCheckRouteResult struct {
	Domain string
	Health HealthCheckData
}

type HealthCheckData struct {
	Ok bool
	Error string
	LastCheck int64
}

func writeError(error string, code int, w http.ResponseWriter) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(HttpError{
		Error: error,
	})
}

func writeSuccess(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(HttpSuccessObject{
		Success: true,
	})
}


func formatHealthCheckStatus(domain string, status *HealthCheckStatus) HealthCheckRouteResult {
	var result HealthCheckRouteResult
	if status == nil || status.LastCheck.IsZero() {
		result = HealthCheckRouteResult{
			Domain: domain,
			Health: HealthCheckData{
				Ok: true,
				Error: status.LastErr,
				LastCheck: 0,
			},
		}
	} else {
		result = HealthCheckRouteResult{
			Domain: domain,
			Health: HealthCheckData{
				Ok: status.IsHealthy,
				Error: status.LastErr,
				LastCheck: status.LastCheck.UTC().Unix(),
			},
		}
	}

	return result
}
