package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

//go:embed ratelimit.lua
var rateLimitScript string

var (
	rdb        *redis.Client
	limitScript *redis.Script
	backendURL  *url.URL
	mlServiceURL string

	reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_requests_total",
		Help: "Total requests seen by the gateway",
	}, []string{"outcome"}) // allowed | blocked

	mlLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gateway_ml_score_latency_seconds",
		Help:    "Latency added by the ML scoring hop",
		Buckets: prometheus.DefBuckets,
	})

	botScoreGauge = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gateway_bot_score",
		Help:    "Distribution of ML bot scores (0=human, 1=bot)",
		Buckets: prometheus.LinearBuckets(0, 0.1, 11),
	})
)

const (
	windowMs    = 60000 // 1 minute sliding window
	baseLimit   = 100   // requests per window for a fully trusted client
	botScoreCut = 0.75  // score above this is treated as high-risk
)

type scoreRequest struct {
	ClientIP      string  `json:"client_ip"`
	UserAgent     string  `json:"user_agent"`
	IntervalMs    float64 `json:"interval_ms"`
	HeaderCount   int     `json:"header_count"`
}

type scoreResponse struct {
	BotScore float64 `json:"bot_score"`
}

func main() {
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	backendRaw := getenv("BACKEND_URL", "http://localhost:9000")
	mlServiceURL = getenv("ML_SERVICE_URL", "http://localhost:8000/score")

	var err error
	backendURL, err = url.Parse(backendRaw)
	if err != nil {
		log.Fatalf("invalid BACKEND_URL: %v", err)
	}

	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	limitScript = redis.NewScript(rateLimitScript)

	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, proxy)
	})

	log.Println("gateway listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleRequest(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	ctx := context.Background()
	clientIP := clientIPFrom(r)

	// 1. Score the request with the ML service (best-effort; fail open on error)
	trustMultiplier := 1.0
	start := time.Now()
	score, err := scoreClient(clientIP, r)
	mlLatency.Observe(time.Since(start).Seconds())
	if err == nil {
		botScoreGauge.Observe(score)
		if score >= botScoreCut {
			trustMultiplier = 0.2 // squeeze flagged clients to 20% of normal limit
		}
	} else {
		log.Printf("ml scoring failed (failing open): %v", err)
	}

	// 2. Check rate limit in Redis (atomic Lua script)
	key := "rl:" + clientIP
	res, err := limitScript.Run(ctx, rdb, []string{key},
		time.Now().UnixMilli(), windowMs, baseLimit, trustMultiplier).Result()
	if err != nil {
		log.Printf("redis rate limit error, failing open: %v", err)
		proxy.ServeHTTP(w, r)
		return
	}

	arr := res.([]interface{})
	allowed := arr[0].(int64) == 1

	if !allowed {
		reqTotal.WithLabelValues("blocked").Inc()
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limit exceeded\n"))
		return
	}

	reqTotal.WithLabelValues("allowed").Inc()
	proxy.ServeHTTP(w, r)
}

func scoreClient(clientIP string, r *http.Request) (float64, error) {
	payload := scoreRequest{
		ClientIP:    clientIP,
		UserAgent:   r.UserAgent(),
		IntervalMs:  0, // populate from a per-client last-seen timestamp in Redis for a real signal
		HeaderCount: len(r.Header),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", mlServiceURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 150 * time.Millisecond} // hard latency budget
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var sr scoreResponse
	if err := json.Unmarshal(data, &sr); err != nil {
		return 0, err
	}
	return sr.BotScore, nil
}

func clientIPFrom(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
