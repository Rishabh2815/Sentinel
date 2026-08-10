# Sentinel

# Sentinel — API gateway with adaptive rate limiting and ML bot detection

A gateway that sits in front of a backend, rate-limits clients using a
distributed sliding-window algorithm backed by Redis, and scores every
request in real time with a lightweight ML model to detect bots — tightening
limits for suspicious clients instead of applying a flat block.

## Why

Most rate limiters treat every client the same. Sentinel differentiates
between a human on a slow connection and a scraper hammering the API at a
fixed interval, and adjusts the limit per client instead of guessing a single
global threshold.

## Architecture

```
client → gateway (Go) ─┬─→ Redis (sliding-window counters)
                        ├─→ ML scorer (FastAPI + scikit-learn)
                        └─→ backend (mock API)

gateway → Prometheus → Grafana (latency, block rate, bot-score distribution)
```

- **Gateway** (`/gateway`): Go reverse proxy. On each request it (1) calls
  the ML scorer for a bot_score, (2) runs an atomic Lua script in Redis to
  check/update the sliding-window counter with a trust-adjusted limit, then
  (3) proxies or rejects with 429.
- **ML scorer** (`/ml-service`): FastAPI service serving a
  `GradientBoostingClassifier` trained on request features (inter-request
  interval, header count, user-agent length, bot-keyword match). Falls back
  to a rule-based heuristic if no trained model is present.
- **Rate limiter**: Redis sorted-set sliding window, atomic via Lua so it's
  safe across multiple gateway replicas.
- **Observability**: Prometheus scrapes gateway metrics (`/metrics`);
  Grafana visualizes request outcome, ML latency overhead, and bot-score
  distribution.
- **Load testing**: k6 scripts simulate both normal browsing traffic and a
  bot swarm to measure throughput, latency, and block accuracy under load.

## Running it

```bash
docker compose up --build
```

- Gateway: `http://localhost:8080`
- ML service: `http://localhost:8000/docs`
- Mock backend: `http://localhost:9000`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (admin/admin)

## Load testing

```bash
k6 run loadtest/normal_traffic.js
k6 run loadtest/bot_swarm.js
```

Run both concurrently to simulate mixed real-world traffic and observe the
block rate / latency split in Grafana.

## Results

*(Fill in after running load tests — this section is what makes the project
credible on a resume. Don't skip it.)*

| Metric | Value |
|---|---|
| Sustained throughput | TBD req/s at pXX ms |
| ML scoring added latency | TBD ms (p50 / p99) |
| Bot traffic blocked | TBD% at TBD% false-positive rate |
| Node failure recovery | TBD (if chaos-tested) |

## Design trade-offs

- **Redis over in-process counters**: enables the gateway to run as multiple
  replicas without inconsistent rate limits — the cost is a network hop per
  request, mitigated by keeping the Lua script atomic and single-round-trip.
- **Fail-open ML scoring**: if the ML service is slow or down, the gateway
  proxies the request rather than blocking — availability over strictness,
  with a hard 150ms timeout so a stalled scorer can't cascade into gateway
  latency.
- **Heuristic fallback**: lets the gateway function correctly before a model
  is trained, so Tier 1 (rate limiting) ships independently of Tier 2 (ML).

## Retraining

`ml-service/train.py` currently trains on synthetic data. To retrain on real
traffic, replace `generate_synthetic_data()` with features pulled from the
Redis Streams event log (see `gateway/main.go` for where to add stream
writes), then rebuild the ml-service image.

## Roadmap / stretch goals

- [ ] Kafka/Redis Streams event log wired into the gateway for full audit trail
- [ ] Chaos test: kill a gateway container mid-load-test, document recovery
- [ ] Adaptive trust decay: flagged clients regain trust over time instead of
      staying permanently throttled
