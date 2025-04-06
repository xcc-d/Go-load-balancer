package proxy

import (
	"context"
	"fmt"
	"go-load-balancer/internal/algorithms"
	"go-load-balancer/internal/backend"
	"go-load-balancer/internal/stats"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// ReverseProxy 实现反向代理
type ReverseProxy struct {
	backendPool    *backend.Pool
	proxy          *httputil.ReverseProxy
	algorithm      algorithms.Algorithm
	statsCollector stats.StatsCollector
}

// NewReverseProxy 创建新的反向代理实例
func NewReverseProxy(pool *backend.Pool, algorithm algorithms.Algorithm, collector stats.StatsCollector) *ReverseProxy {
	rp := &ReverseProxy{
		backendPool:    pool,
		algorithm:      algorithm,
		statsCollector: collector,
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		DisableKeepAlives:     false,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	rp.proxy = &httputil.ReverseProxy{
		Director:       rp.director,
		ModifyResponse: rp.modifyResponse,
		ErrorHandler:   rp.errorHandler,
		Transport:      transport,
		FlushInterval:  -1, // 禁用自动刷新
	}

	return rp
}

// ServeHTTP 实现http.Handler接口
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 设置请求上下文(用于跟踪)
	ctx := r.Context()
	reqID := fmt.Sprintf("%v", time.Now().UnixNano())
	ctx = context.WithValue(ctx, "req_id", reqID)
	ctx = context.WithValue(ctx, "start_time", startTime)
	r = r.WithContext(ctx)

	// 设置请求到算法中(针对IP哈希等需要请求信息的算法)
	rp.algorithm.SetRequest(r)

	// 调用代理
	rp.proxy.ServeHTTP(w, r)
}

// director 修改请求以发送到后端
func (rp *ReverseProxy) director(req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// 使用负载均衡算法获取下一个后端
	peer := rp.algorithm.GetNextBackend()
	if peer == nil {
		log.Printf("无可用后端服务器")
		return
	}

	if !peer.IsAlive() {
		log.Printf("后端服务器 %s 不可用", peer.URL.Host)
		return
	}

	req.URL.Scheme = peer.URL.Scheme
	req.URL.Host = peer.URL.Host
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)

	// 记录后端信息到上下文，以便后续处理
	ctx = context.WithValue(req.Context(), "backend", peer.URL.Host)
	req = req.WithContext(ctx)

	// 增加连接计数
	peer.IncrementConnections()
}

// modifyResponse 修改来自后端的响应
func (rp *ReverseProxy) modifyResponse(res *http.Response) error {
	// 如果请求为空，直接返回
	if res == nil || res.Request == nil {
		return nil
	}

	// 获取请求开始时间
	var startTime time.Time
	if v := res.Request.Context().Value("start_time"); v != nil {
		if t, ok := v.(time.Time); ok {
			startTime = t
		}
	}

	// 获取后端信息
	var backendHost string
	if v := res.Request.Context().Value("backend"); v != nil {
		if s, ok := v.(string); ok {
			backendHost = s
		}
	}

	// 减少后端连接数并记录统计信息
	if backendURL, err := url.Parse(res.Request.URL.String()); err == nil {
		if peers := rp.backendPool.GetBackends(); peers != nil {
			for _, p := range peers {
				if p.URL.Host == backendURL.Host {
					p.DecrementConnections()

					// 记录请求统计
					if rp.statsCollector != nil && startTime != (time.Time{}) {
						duration := time.Since(startTime)
						rp.statsCollector.RecordRequest(backendHost, res.StatusCode, res.Request.Method, duration)
					}

					break
				}
			}
		}
	}
	return nil
}

// errorHandler 处理代理错误
func (rp *ReverseProxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("代理错误: %v", err)

	// 确保释放后端连接
	if backendURL, err := url.Parse(r.URL.String()); err == nil {
		if peers := rp.backendPool.GetBackends(); peers != nil {
			for _, p := range peers {
				if p.URL.Host == backendURL.Host {
					p.DecrementConnections()

					// 记录错误
					if rp.statsCollector != nil {
						backendHost := p.URL.Host
						rp.statsCollector.RecordError(backendHost, "proxy_error")
					}

					break
				}
			}
		}
	}

	// 返回错误响应
	w.WriteHeader(http.StatusBadGateway)
	io.WriteString(w, "Bad Gateway")
}
