package gateway

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
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

	// 타임아웃 / 커넥션 정책
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		// TCP 연결
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,  // TCP 연결시도 최대 대기 시간 (타임아웃)
			KeepAlive: 30 * time.Second, // TCP 연결을 얼마나 유지할지 (연결 생존 여부)
		}).DialContext,

		// HTTP 커넥션풀(재사용) 설정
		MaxIdleConns:    100,              // 커넥션 풀 크기
		IdleConnTimeout: 60 * time.Second, // idle 연결 유지 시간 (재사용 가능 시간)

		// TLS, 프로토콜 단계 타임아웃
		TLSHandshakeTimeout:   2 * time.Second, // TLS 핸드셰이크 최대 대기 시간
		ExpectContinueTimeout: 1 * time.Second, // 100-continue 응답 대기 시간 (POST 대용량 업로드 시)

		// 업스트림 응답 지연 방어
		ResponseHeaderTimeout: 3 * time.Second, // 연결 후 응답 헤더를 받기까지의 최대 대기 시간
	}
	proxy.Transport = transport

	// 업스트림 장애 시 응답 통일 + 로그 남기기 가능
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		// 로그: 어떤 업스트림으로, 어떤 요청이 실패했는지 추적 가능
		log.Printf("[gateway] upstream error: method=%s path=%s target=%s err=%v",
			r.Method, r.URL.Path, targetBaseURL, e)

		status := statusFromUpstreamErr(e)

		// 클라이언트 응답: JSON 통일
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)

		if status == http.StatusGatewayTimeout {
			_, _ = w.Write([]byte(`{"error":"upstream timeout"}`))
			return
		}

		_, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}, nil
}

// 에러 구분
func statusFromUpstreamErr(err error) int {
	// nil 가드 (nil일 경우 502)
	if err == nil {
		return http.StatusBadGateway
	}

	// context deadline(타임아웃)
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout // 504
	}

	// net.Error 중 timeout 계열 (dial/read/write/response header timeout 등)
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return http.StatusGatewayTimeout // 504
	}

	// fallback : 리버스 프록시가 감싼 timeout 문자열 대응
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return http.StatusGatewayTimeout
	}

	// 그 외: 업스트림 연결 실패/리졸브 실패/리셋 등 -> 502
	return http.StatusBadGateway
}
