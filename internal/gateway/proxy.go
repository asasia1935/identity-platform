package gateway

import (
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

		// forward client info (optional but handy)
		if r.Header.Get("X-Forwarded-Proto") == "" {
			r.Header.Set("X-Forwarded-Proto", "http")
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}, nil
}
