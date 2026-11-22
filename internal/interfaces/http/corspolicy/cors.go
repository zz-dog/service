// Package corspolicy 是全项目唯一的跨域策略源:放行哪些来源、放行哪些方法与请求头。
// gin 服务(router 包)经 gin-contrib/cors 使用这份策略;网关是纯 net/http,
// 用这里的 Middleware 直接套在转发逻辑外面。两处共用同一份数据,改一处全生效,
// 避免出现"业务服务放行了某个头、网关没放行,预检莫名失败"这类两份配置漂移的坑。
//
// 网关为什么也要答 CORS:上游服务挂掉时,网关自己写出的 502/503/404 不带跨域头,
// 浏览器会把真实错误伪装成"预检没有 Access-Control-Allow-Origin"的 CORS 报错,
// 排查方向全错;预检在网关就地应答后,即使上游全挂,前端也能看到真实的 502。
package corspolicy

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HeaderMerchantID 商户端"当前店铺"上下文头:N:N 下一个用户可归属多家店,
// 请求必须显式声明操作哪家店。它必须进跨域白名单,否则商户端的预检
// 直接死在浏览器侧,请求根本到不了业务代码。定义放在这里是因为
// 白名单(AllowHeaders)要引用它,而白名单必须全局唯一。
const HeaderMerchantID = "X-Merchant-Id"

// MaxAge 预检结果缓存时长:浏览器在缓存期内不再重复发预检。
const MaxAge = 12 * time.Hour

// AllowMethods 实际暴露给前端的方法集合。
var AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

// AllowHeaders 预检放行的请求头。必须覆盖业务实际使用的全部自定义头。
var AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", HeaderMerchantID}

// ExposeHeaders 允许前端 JS 读到的响应头。
var ExposeHeaders = []string{"Content-Length"}

// IsAllowedOrigin 判断来源是否允许：localhost / 回环 / 私有网段 IP（任意端口）。
// 便于本地与局域网跨端联调；公网域名上线后在这里加白名单。
func IsAllowedOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" || host == "::1" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}
	return false
}

// Middleware 返回 net/http 版跨域中间件,供网关(非 gin)使用。
// 行为与 gin-contrib/cors 对齐:
//   - 无 Origin 头或 Origin 即本站(同源 fetch 也会带 Origin)→ 原样放行;
//   - 来源不在白名单 → 403,不带任何放行头;
//   - 预检(OPTIONS + Access-Control-Request-Method)→ 网关就地应答 204,不再转发上游;
//   - 普通跨域请求 → 转发上游,响应若未带放行头(网关错误响应、上游没配)则补上;
//     上游已答就不重复加——浏览器遇到两个 Access-Control-Allow-Origin 会直接拒绝。
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || origin == "http://"+r.Host || origin == "https://"+r.Host {
			next.ServeHTTP(w, r) // 非跨域请求,CORS 中间件全程不掺手
			return
		}
		if !IsAllowedOrigin(origin) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			answerPreflight(w, origin)
			return
		}
		w.Header().Add("Vary", "Origin")
		next.ServeHTTP(&corsWriter{ResponseWriter: w, origin: origin}, r)
	})
}

// answerPreflight 就地应答预检:204 + 放行头,不转发上游。
// 这正是"上游挂了 CORS 也跟着挂"的解法——预检的成功与否不再依赖上游存活。
// AllowCredentials=true 时 Allow-Origin 必须回显具体来源,不能用 "*";
// 预检结果按请求特征缓存,必须 Vary,防止中间缓存把 A 来源的应答错发给 B。
func answerPreflight(w http.ResponseWriter, origin string) {
	h := w.Header()
	h.Add("Vary", "Origin")
	h.Add("Vary", "Access-Control-Request-Method")
	h.Add("Vary", "Access-Control-Request-Headers")
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Access-Control-Allow-Methods", strings.Join(AllowMethods, ","))
	h.Set("Access-Control-Allow-Headers", strings.Join(AllowHeaders, ","))
	h.Set("Access-Control-Max-Age", strconv.Itoa(int(MaxAge.Seconds())))
	w.WriteHeader(http.StatusNoContent)
}

// corsWriter 在真正写出状态行之前补齐跨域响应头。
// ReverseProxy 在转发时会先拷贝上游响应头再 WriteHeader,而头一旦写出就不可再改,
// 所以必须拦在 WriteHeader 这一步判断:上游答了就只放行,没答(网关 502/503/404、
// 上游没配 CORS)才补上。
type corsWriter struct {
	http.ResponseWriter
	origin      string
	wroteHeader bool
}

// Unwrap 交给 http.ResponseController 用,ReverseProxy 流式转发依赖它找到真实 Flush 能力。
func (cw *corsWriter) Unwrap() http.ResponseWriter { return cw.ResponseWriter }

func (cw *corsWriter) WriteHeader(code int) {
	if cw.wroteHeader {
		return
	}
	cw.wroteHeader = true
	h := cw.Header()
	if h.Get("Access-Control-Allow-Origin") == "" {
		h.Add("Vary", "Origin")
		h.Set("Access-Control-Allow-Origin", cw.origin)
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Set("Access-Control-Expose-Headers", strings.Join(ExposeHeaders, ","))
	}
	cw.ResponseWriter.WriteHeader(code)
}
