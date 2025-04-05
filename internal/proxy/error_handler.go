package proxy

import (
	"log"
	"net/http"
)

// ErrorHandler 处理代理错误
type ErrorHandler struct {
	proxy *ReverseProxy
}

// NewErrorHandler 创建新的错误处理器
func NewErrorHandler(proxy *ReverseProxy) *ErrorHandler {
	return &ErrorHandler{
		proxy: proxy,
	}
}

// HandleError 处理代理错误
func (eh *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("代理请求错误: %v", err)

	switch err {
	case ErrNoAvailableBackend:
		http.Error(w, "服务暂时不可用", http.StatusServiceUnavailable)
	case ErrBackendTimeout:
		http.Error(w, "后端服务响应超时", http.StatusGatewayTimeout)
	default:
		http.Error(w, "网关错误", http.StatusBadGateway)
	}
}

// 定义常见错误
var (
	ErrNoAvailableBackend = NewProxyError("没有可用的后端服务器")
	ErrBackendTimeout     = NewProxyError("后端服务响应超时")
)

// ProxyError 自定义代理错误
type ProxyError struct {
	msg string
}

// NewProxyError 创建新的代理错误
func NewProxyError(msg string) *ProxyError {
	return &ProxyError{msg: msg}
}

func (e *ProxyError) Error() string {
	return e.msg
}
