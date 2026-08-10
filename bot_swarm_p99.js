import http from "k6/http";
import { sleep, check } from "k6";

// Simulates scraper/credential-stuffing bots: high fixed rate,
// near-zero interval variance, scripted user agents.
export const options = {
  scenarios: {
    bot_swarm: {
      executor: "constant-arrival-rate",
      rate: 200, // requests per second
      timeUnit: "1s",
      duration: "1m",
      preAllocatedVUs: 100,
      maxVUs: 300,
    },
  },
  thresholds: {
    http_req_duration: ["p(99)<600"],
  },
};

const BOT_UAS = ["python-requests/2.31", "curl/8.4.0", "Scrapy/2.11", "Go-http-client/1.1"];

export default function () {
  const ua = BOT_UAS[Math.floor(Math.random() * BOT_UAS.length)];
  const res = http.get("http://localhost:8080/api/products", {
    headers: { "User-Agent": ua }, // deliberately sparse headers
  });
  check(res, { "got a response": (r) => r.status !== 0 });
  sleep(0.05); // near-zero, regular interval — the bot tell
}
