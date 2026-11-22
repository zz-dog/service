package corspolicy

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

// 以下用例守护网关跨域行为的四个关键点:
// 1) 预检在网关就地应答——上游挂了预检也必须过(这正是线上 502 被浏览器
//    伪装成 CORS 报错的场景);
// 2) 非白名单来源一律 403,不放行头;
// 3) 网关自己的错误响应(502/503/404)要带放行头,否则前端永远看到的是误导性 CORS 报错;
// 4) 上游已答 CORS 时绝不重复加——响应里出现两个 Access-Control-Allow-Origin
//    浏览器会直接拒绝。

// newNext 构造被包一层的下游 handler:模拟"网关错误路径"或"上游已答 CORS"两种情况。
func newNext(status int, setUpstreamCORS bool, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if setUpstreamCORS {
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

func TestPreflightAnsweredLocallyEvenWhenUpstreamDown(t *testing.T) {
	reached := false
	// 模拟上游彻底挂掉:只要被调用就算测试失败
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })
	h := Middleware(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/merchant/shops", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "authorization,x-merchant-id")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if reached {
		t.Fatal("预检不应转发到下游:上游挂掉时预检也必须成功")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("预检应答 204,得到 %d", rec.Code)
	}
	hdr := rec.Header()
	if got := hdr.Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
		t.Errorf("Allow-Origin 应回显具体来源(AllowCredentials 不能用 *),得到 %q", got)
	}
	if got := hdr.Get("Access-Control-Allow-Headers"); !strings.Contains(got, HeaderMerchantID) {
		t.Errorf("Allow-Headers 必须包含商户上下文头 %s,得到 %q", HeaderMerchantID, got)
	}
	if got := hdr.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials 应为 true,得到 %q", got)
	}
	if got := hdr.Get("Access-Control-Max-Age"); got != "43200" {
		t.Errorf("MaxAge 应为 43200 秒,得到 %q", got)
	}
	if !strings.Contains(hdr.Get("Vary"), "Origin") {
		t.Errorf("预检按来源缓存,必须 Vary: Origin")
	}
}

func TestPreflightRejectedForDisallowedOrigin(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("非白名单来源不应到达下游")
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/user/info", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("非白名单来源应 403,得到 %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("403 响应不应带放行头,得到 %q", got)
	}
}

func TestGatewayErrorResponseGetsCORSHeaders(t *testing.T) {
	// 模拟网关自身错误路径(writeError):上游没参与,响应天生没有 CORS 头
	h := Middleware(newNext(http.StatusBadGateway, false, `{"code":502,"message":"上游服务不可用"}`))
	req := httptest.NewRequest(http.MethodGet, "/api/merchant/shops", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("应透传 502,得到 %d", rec.Code)
	}
	hdr := rec.Header()
	if got := hdr.Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
		t.Errorf("网关错误响应必须补放行头,否则浏览器把 502 报成 CORS 错,得到 %q", got)
	}
	if len(hdr.Values("Access-Control-Allow-Origin")) != 1 {
		t.Errorf("Allow-Origin 只能出现一次")
	}
	if got := hdr.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials 应为 true,得到 %q", got)
	}
}

func TestNoDuplicateHeadersWhenUpstreamAnswersCORS(t *testing.T) {
	h := Middleware(newNext(http.StatusOK, true, "ok"))
	req := httptest.NewRequest(http.MethodGet, "/api/user/info", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	hdr := rec.Header()
	if n := len(hdr.Values("Access-Control-Allow-Origin")); n != 1 {
		t.Fatalf("上游已答 CORS 时网关不得再加,Allow-Origin 出现 %d 次", n)
	}
	if got := hdr.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials 应保持上游值,得到 %q", got)
	}
}

func TestNonCORSRequestPassesThroughUntouched(t *testing.T) {
	h := Middleware(newNext(http.StatusOK, false, "ok"))

	// 无 Origin 头:curl/服务间调用
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("无 Origin 的请求不应被加跨域头,得到 %q", got)
	}

	// 同源 fetch 也会带 Origin,Origin 即本站时按非跨域放行
	req := httptest.NewRequest(http.MethodGet, "http://139.196.114.185:8080/api/user/info", nil)
	req.Header.Set("Origin", "http://139.196.114.185:8080")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if got := rec2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("同源请求不应被加跨域头,得到 %q", got)
	}
}

func TestReverseProxyStreamingThroughCorsWriter(t *testing.T) {
	// corsWriter 拦截了 WriteHeader,必须保住 ReverseProxy 流式转发依赖的 Flush 能力
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	proxy := &httputil.ReverseProxy{Rewrite: func(pr *httputil.ProxyRequest) { pr.SetURL(target) }}

	h := Middleware(proxy)
	req := httptest.NewRequest(http.MethodGet, "/api/any", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("经 ReverseProxy 转发应得到 200,得到 %d", rec.Code)
	}
	if n := len(rec.Header().Values("Access-Control-Allow-Origin")); n != 1 {
		t.Errorf("转发响应应补且仅补一次 Allow-Origin,出现 %d 次", n)
	}
}
