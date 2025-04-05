package proxy

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"Go-load-balancer/internal/backend"
)

// ReverseProxy 实现反向代理
type ReverseProxy struct {
	backendPool *backend.Pool
	proxy       *httputil.ReverseProxy
}

// NewReverseProxy 创建新的反向代理实例
func NewReverseProxy(pool *backend.Pool) *ReverseProxy {
	rp := &ReverseProxy{
		backendPool: pool,
	}

	rp.proxy = &httputil.ReverseProxy{
		Director:       rp.director,
		ModifyResponse: rp.modifyResponse,
		ErrorHandler:   rp.errorHandler,
	}

	return rp
}

// ServeHTTP 实现http.Handler接口
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp.proxy.ServeHTTP(w, r)
}

// director 修改请求以发送到后端
func (rp *ReverseProxy) director(req *http.Request) {
	peer, err := rp.backendPool.GetNextPeer()
	if err != nil {
		log.Fatal(err)
	}

	req.URL.Scheme = peer.URL.Scheme
	req.URL.Host = peer.URL.Host
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)

	peer.IncrementConnections()
}

// modifyResponse 修改来自后端的响应
func (rp *ReverseProxy) modifyResponse(res *http.Response) error {
	// 减少后端连接数
	if backendURL, err := url.Parse(res.Request.URL.String()); err == nil {
		if peers := rp.backendPool.GetBackends(); peers != nil {
			for _, p := range peers {
				if p.URL.Host == backendURL.Host {
					p.DecrementConnections()
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
	w.WriteHeader(http.StatusBadGateway)
	io.WriteString(w, "Bad Gateway")
}
