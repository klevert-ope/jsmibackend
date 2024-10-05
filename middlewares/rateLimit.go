package middlewares

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type RateLimiter struct {
	limits     sync.Map
	limit      int
	window     time.Duration
	cleanupInt time.Duration
}

type clientData struct {
	requests int32
	timer    *time.Timer
}

func NewRateLimiter(limit int, window time.Duration, cleanupInt time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limit:      limit,
		window:     window,
		cleanupInt: cleanupInt,
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) SetLimit(limit int) {
	rl.limit = limit
}

func (rl *RateLimiter) SetWindow(window time.Duration) {
	rl.window = window
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(rl.cleanupInt)
		rl.limits.Range(func(key, value interface{}) bool {
			data := value.(*clientData)
			if atomic.LoadInt32(&data.requests) == 0 {
				data.timer.Stop()
				rl.limits.Delete(key)
				//log.Printf("Cleaned up rate limiter entry for %v", key)
			}
			return true
		})
	}
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				return ip
			}
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	if parsedIP := net.ParseIP(ip); parsedIP != nil {
		return ip
	}
	return ""
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		data, _ := rl.limits.LoadOrStore(clientIP, &clientData{
			requests: 0,
			timer: time.AfterFunc(rl.window, func() {
				rl.resetRequests(clientIP)
			}),
		})
		clientData := data.(*clientData)

		if atomic.AddInt32(&clientData.requests, 1) > int32(rl.limit) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			//log.Printf("Blocked request from %s due to rate limiting", clientIP)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) resetRequests(clientIP string) {
	data, ok := rl.limits.Load(clientIP)
	if !ok {
		return
	}
	clientData := data.(*clientData)
	atomic.StoreInt32(&clientData.requests, 0)
	clientData.timer.Reset(rl.window)
}
