# Web Analyzer

A web page analysis tool built with Go. You give it a URL, it fetches the page and tells you everything useful about it — HTML version, heading structure, internal vs external links, broken links, login form detection, and more.

## System Design

![System Design](SystemDesign.png)

## What It Does

When you submit a URL, the analyzer:

1. Fetches the page using an HTTP client with a 20-second timeout
2. Parses the HTML and extracts:
   - HTML version (HTML5, HTML 4.01, XHTML, etc.)
   - Page title
   - Heading counts (h1 through h6)
   - Internal and external link counts
   - Whether a login form is present
3. Concurrently checks all links (up to 10 at a time) to find broken/inaccessible ones
4. Returns the results rendered in a clean HTML page

---

## Demo

<video src="https://raw.githubusercontent.com/Pirathikaran/web-analyzer-go/master/demo.mp4" controls width="100%"></video>

---

## Project Structure

```
web-analyzer-go/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── analyzer/            # Core analysis logic
│   ├── handler/             # HTTP handlers and middleware
│   └── metrics/             # Prometheus metrics
├── web/
│   └── templates/
│       └── index.html       # UI template
├── Dockerfile
├── Makefile
└── go.mod
```

---

## Running Locally

### Prerequisites

- Go 1.23+

### Steps

```bash
# Clone the repo
git clone https://github.com/Pirathikaran/web-analyzer.git
cd web-analyzer-go

# Download dependencies
go mod download

# Copy the env template and edit as needed
cp .env.example .env

# Run the server
go run ./cmd/server
```

The server starts on **http://localhost:8080**

You can override the port:

```bash
PORT=9090 go run ./cmd/server
```

---

## Running with Docker

No Go installation needed — Docker handles everything.

### Build the image

```bash
docker build -t web-analyzer .
```

### Run the container

```bash
docker run -p 8080:8080 web-analyzer
```

Then open **http://localhost:8080** in your browser.

### Use a different port

```bash
docker run -p 3000:3000 -e PORT=3000 web-analyzer
```

---

## API Endpoints

| Method | Path       | Description                          |
|--------|------------|--------------------------------------|
| GET    | `/`        | Home page with URL input form        |
| POST   | `/analyze` | Submit a URL and get analysis result |
| GET    | `/metrics` | Prometheus metrics endpoint          |

---

## Observability

### Prometheus Metrics

Available at `GET /metrics`:

| Metric                                    | Type      | Description                        |
|-------------------------------------------|-----------|------------------------------------|
| `web_analyzer_requests_total`             | Counter   | Total requests by status           |
| `web_analyzer_request_duration_seconds`   | Histogram | Request latency distribution       |
| `web_analyzer_errors_total`               | Counter   | Total analysis errors              |

### pprof (Debug Profiling)

A pprof debug server runs on `127.0.0.1:6060` (localhost only). Useful for profiling in development:

```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## Running Tests

```bash
go test ./...
```

---

## Configuration

| Environment Variable | Default | Description                                              |
|----------------------|---------|----------------------------------------------------------|
| `PORT`               | `8080`  | HTTP listen port                                         |
| `POOL_WORKERS`       | `20`    | Number of concurrent analysis workers                    |
| `POOL_QUEUE`         | `1000`  | Max queued analysis requests before returning 503        |
| `LINK_SEM`           | `50`    | Max concurrent outbound link-check HTTP requests         |

### Tuning Guide

- **`POOL_WORKERS`** — increase if you have spare CPU/network capacity and see high queue wait times; decrease to reduce outbound load.
- **`POOL_QUEUE`** — buffer for bursts. If requests arrive faster than workers process them and the queue fills, clients receive a `503 Server Busy`. Size this to your expected burst length.
- **`LINK_SEM`** — global cap on simultaneous outbound HEAD/GET requests across all workers. Prevents overwhelming target servers or exhausting local file descriptors. Rule of thumb: `POOL_WORKERS × 10` is the uncapped ceiling; `LINK_SEM` keeps it sane.

### Setup

Copy the template and edit values for your environment:

```bash
cp .env.example .env
```

`.env` is gitignored — never committed. `.env.example` is committed as the canonical reference.

> **Note:** Go does not auto-load `.env` files. Use a process manager (Docker, systemd, Make) to source them, or export manually:
> ```bash
> export $(grep -v '^#' .env | xargs) && go run ./cmd/server
> ```

### Example: high-throughput deployment

```bash
POOL_WORKERS=40 POOL_QUEUE=2000 LINK_SEM=100 ./server
```

### Example: low-resource / rate-limited environment

```bash
POOL_WORKERS=5 POOL_QUEUE=200 LINK_SEM=20 ./server
```

---

## Tech Stack

- **Go 1.23** — standard library HTTP server, structured logging (`slog`)
- **golang.org/x/net/html** — HTML parsing
- **Prometheus** — metrics and monitoring
- **Docker** — containerized deployment (multi-stage build, minimal Alpine image)

---

## Future Improvements

- **Per-IP Rate Limiting** — Current rate limiting is global. It can be improved by implementing per-IP or per-user rate limits to prevent a single client from consuming all available requests.

- **Caching Layer** — Introduce a caching mechanism using Redis to store analysis results for frequently requested URLs. This reduces repeated processing and improves response time.

- **Horizontal Scalability** — Deploy multiple instances of the service behind a load balancer so the system can handle higher traffic and scale efficiently.

- **Security Improvements** — Add HTTP security headers and implement additional validation to prevent attacks such as Server-Side Request Forgery (SSRF).

- **Observability Enhancements** — Extend monitoring by adding dashboards with Grafana and perform load testing using k6 to evaluate performance under heavy traffic.
