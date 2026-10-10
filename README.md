# Sentinel

A distributed API gateway written in Go that rate-limits traffic with Redis and scores requests for bot-likelihood using a Python ML microservice.

## Components

| Component | Role |
|---|---|
| Gateway (Go) | Reverse proxy; applies rate limiting, calls the scorer, blocks or forwards |
| Redis | Backing store for atomic sliding-window rate limiting |
| ML scorer (Python, FastAPI, scikit-learn) | Returns a bot score for a request |
| Prometheus + Grafana | Metrics and dashboards |
| Docker | Containerised local deployment |

> Every number in this README is labelled **Measured**, **Estimated** or **Not verified**. If a result is not backed by a saved artifact in `results/`, it does not appear as a measured claim.

---

## Benchmark results

All figures below come from a single k6 run. Full conditions are in [Test evidence](#test-evidence).

| Metric | Value | Status |
|---|---|---|
| Bot traffic blocked | **99.16%** | Measured (k6) |
| p95 latency | **< 16 ms** | Measured (k6) |
| p50 latency | `<FILL from k6 summary>` | Measured |
| p99 latency | `<FILL from k6 summary>` | Measured |
| Throughput (req/s) | `<FILL: http_reqs rate>` | Measured |
| Legitimate traffic wrongly blocked (false positives) | `<FILL: %>` | Measured |
| Error rate (5xx / timeouts) | `<FILL: http_req_failed>` | Measured |

If you cannot fill a row from the saved k6 output, delete the row rather than leave a placeholder.

### Test conditions

| Item | Value |
|---|---|
| Date of run | `<FILL>` |
| Machine (CPU / RAM / OS) | `<FILL>` |
| Deployment | Docker Compose on a single machine (`<FILL: confirm>`) |
| Gateway instances | `<FILL>` |
| Redis | `<FILL: version, single node, persistence on/off>` |
| ML scorer | `<FILL: workers, model file / version>` |
| k6 runs on same host as the system under test | `<FILL: yes/no>` (if yes, say so; it affects latency) |
| Upstream/backend behind the gateway | `<FILL: real service or stub, and its response time>` |

---

## Test evidence

### Command

```bash
# Replace with the exact command that produced the numbers above
k6 run \
  --summary-export=results/k6-summary.json \
  <FILL: script path, e.g. loadtest/bot_vs_human.js>
```

Pin the k6 version used: `<FILL: k6 version>`.

### Workload

| Item | Value |
|---|---|
| Script | `<FILL path>` (committed to the repo) |
| Traffic mix | `<FILL: e.g. N% bot-labelled, M% legitimate>` |
| How bot traffic is generated | `<FILL: user agents, request rate per client, header patterns>` |
| How legitimate traffic is generated | `<FILL>` |
| Endpoints hit | `<FILL>` |
| Scenario / executor | `<FILL: constant-vus, ramping-vus, constant-arrival-rate>` |

### Concurrency and volume

| Item | Value |
|---|---|
| Virtual users (VUs) | `<FILL>` |
| Duration | `<FILL>` |
| Total requests | `<FILL: http_reqs count>` |
| Bot-labelled requests | `<FILL>` |
| Legitimate requests | `<FILL>` |

### How "bot traffic blocked" is calculated

```
bot_blocked_pct = (bot-labelled requests that received a block response)
                  / (all bot-labelled requests sent) x 100
```

- A **block response** is `<FILL: status code, e.g. 403 or 429>` returned by the gateway.
- Requests are labelled as bot or legitimate **by the k6 script** (via a tag or custom counter), not by the gateway's own decision. This keeps ground truth independent of the system being measured.
- The 99.16% figure is `<FILL: blocked count> / <FILL: bot request count>`.
- Blocks from the rate limiter and blocks from the ML scorer are `<FILL: counted together / reported separately>`.

Suggested k6 counters so the number is reproducible:

```javascript
import { Counter } from 'k6/metrics';
const botSent = new Counter('bot_sent');
const botBlocked = new Counter('bot_blocked');
const humanSent = new Counter('human_sent');
const humanBlocked = new Counter('human_blocked');
```

### Artifacts

- `results/k6-summary.json` (raw k6 output)
- `results/RESULTS.md` (dated run log)
- Grafana screenshot or Prometheus export for the run window: `<FILL path>`

---

## Failure handling

This section separates what has been **tested** (with evidence) from what is **intended** (designed but not exercised). Update the status column only when a test has been run and its output saved.

| Failure | Intended behaviour | Status | Evidence |
|---|---|---|---|
| ML scorer times out | `<FILL: fail-open (forward) or fail-closed (block)>` after `<FILL: timeout ms>` | `<Tested / Intended only>` | `<FILL: test name + output, or "none">` |
| ML scorer unreachable or returns an error | `<FILL>` | `<Tested / Intended only>` | `<FILL>` |
| Redis unavailable at request time | `<FILL: fail-open, fail-closed, or in-memory fallback>` | `<Tested / Intended only>` | `<FILL>` |
| Redis recovers after an outage | `<FILL: does the limiter resume without a gateway restart?>` | `<Tested / Intended only>` | `<FILL>` |

### How to test these and record the result

**ML scorer timeout**
1. Add an artificial delay to the scorer longer than the gateway timeout (for example `time.sleep()` in the FastAPI handler).
2. Send requests through the gateway and record the status codes and latency.
3. Record: were requests forwarded or blocked, and what was p95 latency while the scorer was slow?

**Redis unavailable**
1. Start steady load with k6.
2. Run `docker compose stop redis` mid-run, then `docker compose start redis` later.
3. Record: response codes during the outage, whether the gateway crashed, and whether limiting resumed after recovery.

### Write the outcome honestly

- If the behaviour was observed: write "Tested: <what happened>" and link the output.
- If it was only reasoned about from the code: write "Intended, not tested."
- Fail-open and fail-closed both have tradeoffs. Fail-open keeps legitimate traffic flowing but lets bots through during the outage. Fail-closed protects the backend but turns a Redis outage into a full outage. State which one Sentinel chose and why.

---

## Limitations

- Results come from a single-machine run; they are not a production capacity claim.
- The ML model was evaluated on `<FILL: synthetic / labelled dataset>`; accuracy on real-world traffic is Not verified.
- `<FILL: any other known gaps>`

## Running it

```bash
docker compose up --build
# gateway: <FILL port>, Grafana: <FILL port>
```

## Reproducing the benchmark

```bash
docker compose up -d
k6 run --summary-export=results/k6-summary.json <FILL script>
```




