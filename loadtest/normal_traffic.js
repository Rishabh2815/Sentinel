import http from "k6/http";
import { sleep, check } from "k6";

// Simulates realistic human browsing: moderate rate, varied intervals,
// normal browser user agents.
export const options = {
  scenarios: {
    normal_users: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "30s", target: 50 },
        { duration: "1m", target: 50 },
        { duration: "30s", target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<300", "p(99)<600"],
    http_req_failed: ["rate<0.05"],
  },
};

const UAS = [
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0",
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Safari/605.1.15",
  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/124.0",
];

export default function () {
  const ua = UAS[Math.floor(Math.random() * UAS.length)];
  const res = http.get("http://localhost:8080/api/products", {
    headers: {
      "User-Agent": ua,
      "Accept": "text/html,application/json",
      "Accept-Language": "en-US,en;q=0.9",
    },
  });
  check(res, { "status is 200 or 429": (r) => r.status === 200 || r.status === 429 });
  sleep(Math.random() * 2 + 0.5); // 0.5-2.5s between requests, human-like
}
