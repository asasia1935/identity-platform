package gateway

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewReverseProxyHandler creates a handler that proxies to targetBaseURL
// and rewrites path by stripping `stripPrefix` (e.g. "/api").
func NewReverseProxyHandler(targetBaseURL string, stripPrefix string) (func(http.ResponseWriter, *http.Request), error) {
	target, err := url.Parse(targetBaseURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)

		// strip prefix from path
		if stripPrefix != "" && strings.HasPrefix(r.URL.Path, stripPrefix) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, stripPrefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}

		// 경계 강제용 증표 헤더 (업스트림 서비스가 헤더 없으면 API 거부, 보안적 의도보단 경계 강제 및 아키텍처 의도 표현)
		r.Header.Set("X-Gateway-Verified", "true")

		// forward client info (optional but handy)
		if r.Header.Get("X-Forwarded-Proto") == "" {
			r.Header.Set("X-Forwarded-Proto", "http")
		}
	}

	// 업스트림 장애 시 응답 통일 + 로그 남기기 가능
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		// 로그: 어떤 업스트림으로, 어떤 요청이 실패했는지 추적 가능
		log.Printf("[gateway] upstream error: method=%s path=%s target=%s err=%v",
			r.Method, r.URL.Path, targetBaseURL, e)

		// 클라이언트 응답: JSON 통일
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway) // 502
		_, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}, nil
}
