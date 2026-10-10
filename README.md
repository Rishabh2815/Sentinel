# Sentinel

A distributed API gateway written in Go. It rate-limits clients with Redis (atomic sliding window) and scores requests for bot-likelihood with a Python ML microservice.

## Components

| Component | Role |
|---|---|
| Gateway (Go) | Reverse proxy; applies rate limiting, calls the scorer, blocks or forwards |
| Redis | Backing store for atomic sliding-window rate limiting |
| ML scorer (Python, FastAPI, scikit-learn) | Returns a bot score for a request (gateway timeout budget: 150 ms) |
| Prometheus + Grafana | Metrics and dashboards |
| Docker Compose | Local deployment of the full stack |

> Every number below is labelled **Measured** or **Estimated**. Estimated values were not printed by k6 and should not be quoted as measurements.

---

## Results

### Test environment

| Item | Value |
|---|---|
| Load tool | k6 v0.50.0 |
| OS | Windows 11, run locally |
| Stack under test | Full Docker Compose stack: gateway + Redis + ML service + mock backend |
| Machine (CPU / RAM) | `<FILL>` |
| k6 on same host as the stack | `<FILL: yes/no>` (if yes, results include local contention) |

### Run 1: normal traffic

50 VUs, ramping, human-like request intervals, spoofed per-VU source IPs.

| Metric | Value | Status |
|---|---|---|
| Total requests | 2,944 | Measured |
| Throughput | ~24 req/s | Measured |
| p90 latency | 12.99 ms | Measured |
| p95 latency | 15.19 ms | Measured |
| p99 latency | ~20-25 ms | Estimated (extrapolated from p90/p95; not printed by k6) |
| Max latency | 132.13 ms | Measured |
| Legitimate requests blocked (HTTP 429) | `<FILL: count from k6 output>` | Measured |

### Run 2: bot swarm

200 req/s constant arrival rate, 20 rotating source IPs.

| Metric | Value | Status |
|---|---|---|
| Total requests | 12,001 | Measured |
| Throughput | 199.8 req/s (configured rate) | Measured |
| Blocked (HTTP 429) | 11,901 / 12,001 = 99.17% | Measured |
| p90 latency (all traffic) | 8.48 ms | Measured |
| p95 latency (all traffic) | 159.93 ms | Measured |
| p99 latency (all traffic) | ~200-250 ms | Estimated (not printed by k6) |
| Max latency | 298.58 ms | Measured |

**Note on the two latency figures.** The sub-16 ms p95 comes from Run 1 (normal traffic). The 99.17% blocked figure comes from Run 2 (bot swarm), where p95 was 159.93 ms. They are separate runs and should be quoted separately.

**Note on 99.16% vs 99.17%.** 11,901 / 12,001 = 99.167%. The earlier results report states 99.16% (truncated); rounded to two decimals it is 99.17%.

---

## Test evidence

### Commands

```powershell
# Windows (as run)
k6.exe run loadtest/normal_traffic.js
k6.exe run loadtest/bot_swarm.js
```

To get p99 printed directly by k6 and save raw output:

```bash
k6 run \
  --summary-trend-stats="min,med,avg,p(90),p(95),p(99),max" \
  --summary-export=results/normal_traffic.json \
  loadtest/normal_traffic.js

k6 run \
  --summary-trend-stats="min,med,avg,p(90),p(95),p(99),max" \
  --summary-export=results/bot_swarm.json \
  loadtest/bot_swarm.js
```

Look for `p(99)=` on the `http_req_duration` line. The p99 values in the tables above should be replaced with these once the runs are repeated.

Bring the stack up first:

```bash
docker compose up --build -d
```

### Workloads

| | Normal traffic | Bot swarm |
|---|---|---|
| Script | `loadtest/normal_traffic.js` | `loadtest/bot_swarm.js` |
| Load model | 50 VUs, ramping | 200 req/s, constant arrival rate |
| Client behaviour | Human-like intervals | Flood from 20 rotating source IPs |
| Source IPs | Spoofed, one per VU | 20, rotating |
| Requests | 2,944 | 12,001 |

### How "bot traffic blocked" is calculated

```
blocked_pct = requests answered with HTTP 429 / total requests in the bot-swarm run x 100
            = 11,901 / 12,001
            = 99.167%
```

- Every request in the bot-swarm run is bot traffic by construction, so the denominator is all requests.
- The 429 count comes from k6's `http_req_failed` count.
- It can be cross-checked against the gateway's own counter: `gateway_requests_total{outcome="blocked"}` on the `/metrics` endpoint.
- 100 of 12,001 requests were not blocked (12,001 - 11,901).

### What this run does and does not show

- Blocked requests were rejected quickly (p90 8.48 ms) because Redis denies over-limit clients before the request reaches the ML scorer or the backend.
- The blocked percentage measures **rate-limit enforcement** against a flood. It is not a measure of ML classification accuracy; that is Not verified.
- The normal-traffic run shows p95 under 16 ms with every request being scored by the ML service under a 150 ms timeout budget.

---

## Failure handling

"Tested" means there is saved output showing the behaviour. "Intended" means it is how the code is designed to behave but has not been exercised.

| Failure | Intended behaviour | Status | Evidence |
|---|---|---|---|
| ML scorer exceeds its 150 ms timeout | `<FILL: fail-open (forward) or fail-closed (block)>` | `<Tested / Intended only>` | `<FILL: output, or "none">` |
| ML scorer unreachable or errors | `<FILL>` | `<Tested / Intended only>` | `<FILL>` |
| Redis unavailable | `<FILL: fail-open, fail-closed, or fallback>` | `<Tested / Intended only>` | `<FILL>` |
| Redis recovers after an outage | `<FILL: limiter resumes without gateway restart?>` | `<Tested / Intended only>` | `<FILL>` |

To test:

- **Scorer timeout:** add a delay longer than 150 ms to the FastAPI handler, send requests through the gateway, and record status codes and latency.
- **Redis down:** run steady k6 load, `docker compose stop redis` mid-run, later `docker compose start redis`, and record responses during the outage and after recovery.

State which of fail-open and fail-closed Sentinel uses and why. Fail-open keeps legitimate traffic flowing but lets bots through during the outage; fail-closed protects the backend but turns a Redis outage into a full outage.

---

## Limitations

- Single-machine local run on Windows 11 with Docker; not a production capacity claim.
- p99 values are estimates until the scripts are rerun with p(99) enabled.
- The bot-swarm p95 (159.93 ms) is about 19x its p90 (8.48 ms). The cause of the tail is still being investigated; one hypothesis is requests that reached the ML scorer, whose timeout budget is 150 ms.
- ML bot-detection accuracy on real traffic is Not verified.
- Failure-handling behaviour is documented above as Intended unless marked Tested.






