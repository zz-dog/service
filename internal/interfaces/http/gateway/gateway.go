// Package gateway 实现基于路径前缀路由 + 服务发现的反向代理网关。
package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// Resolver 按服务名解析出一个可用实例地址（ip:port）
type Resolver interface {
	Resolve(serviceName string) (string, error)
}

// Route 转发规则：路径前缀 → 服务名
type Route struct {
	Prefix      string // 如 /api/user
	ServiceName string // 目标服务在 Nacos 中注册的名字
}

// Gateway 反向代理网关：按最长前缀匹配路由，转发前按服务名解析实例地址
type Gateway struct {
	mu       sync.RWMutex
	routes   []Route // 按前缀长度降序，长前缀优先；可通过 UpdateRoutes 热更新
	resolver Resolver
	proxy    *httputil.ReverseProxy
}

func New(routes []Route, resolver Resolver) *Gateway {
	g := &Gateway{resolver: resolver}
	g.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// 目标地址由 ServeHTTP 解析后经 context 传入
			addr, _ := pr.In.Context().Value(targetKey{}).(*string)
			if addr == nil {
				return
			}
			target, err := url.Parse("http://" + *addr)
			if err != nil {
				return
			}
			pr.SetURL(target)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, http.StatusBadGateway, "上游服务不可用")
		},
	}
	g.UpdateRoutes(routes)
	return g
}

// UpdateRoutes 热更新路由表（按前缀长度降序，长前缀优先）；
// 空路由将被忽略，避免误下发的空配置清空全部转发规则
func (g *Gateway) UpdateRoutes(routes []Route) {
	if len(routes) == 0 {
		return
	}
	sorted := append([]Route(nil), routes...)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Prefix) > len(sorted[j].Prefix)
	})
	g.mu.Lock()
	defer g.mu.Unlock()
	g.routes = sorted
}

type targetKey struct{}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 网关自身健康检查
	if r.URL.Path == "/health" {
		writeOK(w)
		return
	}

	serviceName, ok := g.match(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "路由不存在")
		return
	}
	addr, err := g.resolver.Resolve(serviceName)
	if err != nil {
		// 服务在 Nacos 上无健康实例
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	g.proxy.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), targetKey{}, &addr)))
}

// match 最长前缀匹配：/api/user/xxx 命中 /api/user 而非 /api
func (g *Gateway) match(path string) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, route := range g.routes {
		if route.Prefix == "/" || strings.HasPrefix(path, route.Prefix) {
			return route.ServiceName, true
		}
	}
	return "", false
}

// writeJSON 按项目统一响应格式输出 {code, message}
func writeJSON(w http.ResponseWriter, httpStatus, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": msg})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, status, msg)
}

func writeOK(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, 0, "ok")
}
