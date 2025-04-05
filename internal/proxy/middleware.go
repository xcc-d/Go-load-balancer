package proxy

import (
	"log"
	"net/http"
	"time"
)

// Middleware 中间件函数类型
type Middleware func(http.Handler) http.Handler

// LoggingMiddleware 记录请求日志
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(limit int, interval time.Duration) Middleware {
	ticker := time.NewTicker(interval)
	counter := 0

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-ticker.C:
				counter = 0
			default:
				if counter >= limit {
					http.Error(w, "请求过于频繁", http.StatusTooManyRequests)
					return
				}
				counter++
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ChainMiddleware 链式调用多个中间件
func ChainMiddleware(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
