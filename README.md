# PrismDNS

PrismDNS is a lightweight, high-performance authoritative DNS proxy written in Go. It acts as a DNS server that routes queries to different upstream DNS servers based on the domain being queried. This allows multiple DNS servers to share the same public IP address.

## Features

- **Domain-based routing**: Forward DNS queries to different upstream servers based on domain names
- **Fallback routing**: Configurable fallback server when no specific route matches
- **Health checks**: Automatic health monitoring of upstream servers every 5 seconds
- **Rate limiting**: Built-in rate limiting to prevent abuse (configurable per domain or globally)
- **HTTP API**: RESTful API for dynamic route management without restarting the server

## Usage

### Without Docker

Start PrismDNS with your routes:

```bash
./PrismDNS-linux-amd64 \
  -route example.com,10.0.1.5 \
  -route dept.example.com,10.0.2.8 \
  -route api.internal,10.0.1.6 \
  -fallback 10.0.1.7:1053 \
  -http-api-key your-secret-api-key
```

For a full list of flags, run `./PrismDNS-linux-amd64 -help`

### With Docker

```bash
docker run -d \
  -p 53:53/udp \
  -p 53:53/tcp \
  -p 8053:8053 \
  truewinter/prismdns:{version} \
  -route example.com,10.0.1.5 \
  -route dept.example.com,10.0.2.8 \
  -route api.internal,10.0.1.6 \
  -fallback 10.0.1.7:1053 \
  -http-api-key your-secret-api-key
```

Images are not published with the `latest` tag. Check the releases page for the latest version.

For a full list of flags, run `docker run prismdns -help`

## HTTP REST API

All API requests require the `X-Api-Key` header (except for the apex `/healthcheck` and `/metrics` endpoints).

### GET `/routes`
List all configured routes.

**Response:**
```json
[
  {"Domain":"example.com","Ip":"10.0.1.5","Port":53},
  {"Domain":"api.internal","Ip":"10.0.2.8","Port":53}
]
```

### POST `/routes`
Add a new route.

**Request:**
```json
{"Domain":"newdomain.com","Ip":"10.0.3.5","Port":53}
```

### PUT `/routes/{route}`
Modify an existing route.

**Request:**
```json
{"Ip":"example.com","Port":5300}
```

### DELETE `/routes/{route}`
Delete a route.

### GET `/healthcheck/routes`
Get health status of all configured routes.

**Response:**
```json
[
  {
    "Domain":"example.com",
    "Ip": "10.0.1.5",
    "Port": 53,
    "Health": {
      "Ok": true,
      "Error": "",
      "LastCheck": 1779670849,
      "LastCheckLatency": 1.204
    }
  }
]
```

### GET `/healthcheck`
Basic health check endpoint for the HTTP API (no authentication required).

### GET `/healthcheck/routes/{route}`
Get health status of a specific route. See [GET `/healthcheck/routes`](#get-healthcheckroutes).

### GET `/metrics`
Prometheus metrics endpoint (no authentication required). A [pre-built Grafana dashboard](grafana.json) is available for your convenience.

## Chaos Queries

For troubleshooting, PrismDNS supports DNS CHAOS queries for version information:

```bash
# Returns the version
dig @10.0.53.53 CH TXT version.bind

# Returns the server ID set by the server-id flag
dig @10.0.53.53 CH TXT id.server
```

## Building

### Without Docker

Build the binaries for your platform:

```bash
./build.sh
```

This compiles both Linux and Windows executables to the `dist/` directory.

### With Docker

```bash
docker build -t truewinter/prismdns .
```