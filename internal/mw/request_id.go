// internal/mw/request_id.go
package mw

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 헤더용
const RequestIDHeader = "X-Request-Id"

// 컨텍스트용
const CtxKeyRequestID = "request_id"

// 클라 입장에서 제공하는 RequestID를 기본 검증을 패스하면 받아들이고 그렇지 않으면 새로 생성

// 8~64글자 제한
const (
	requestIDMinLen = 8
	requestIDMaxLen = 64
)

// generateRequestID : 16 바이트의 랜덤 값을 32자리의 16진수 값(ID)으로 만들어 반환
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback: keep it unique-ish to avoid collisions in logs
		return "reqid-fallback-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}

// isValidRequestID : RequestID 검증 함수
func isValidRequestID(s string) bool {
	n := len(s)
	if n < requestIDMinLen || n > requestIDMaxLen {
		return false
	}

	for i := 0; i < n; i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '-' || c == '_' || c == '.':
		default:
			// reject spaces, newlines, control chars, etc.
			return false
		}
	}
	return true
}

// RequestIDStrict : 헤더에 리퀘스트 아이디가 없으면 새 아이디를 생성, 있으면 검증 후 세팅 (요청 및 응답 헤더에 반드시 세팅 + 컨텍스트에도 세팅)
func RequestIDStrict() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(RequestIDHeader)

		if rid == "" || !isValidRequestID(rid) {
			rid = generateRequestID()
		}

		c.Set(CtxKeyRequestID, rid)

		// propagate downstream + return to client
		c.Request.Header.Set(RequestIDHeader, rid)
		c.Writer.Header().Set(RequestIDHeader, rid)

		c.Next()
	}
}

// RequestIDPassthrough : 내부 서비스용 (게이트웨이가 이미 엄격히 체크했다는 가정 하에 내부 서비스에서 재검증 X)
func RequestIDPassthrough() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(RequestIDHeader)
		if rid == "" {
			rid = generateRequestID()
		}

		c.Set(CtxKeyRequestID, rid)

		c.Request.Header.Set(RequestIDHeader, rid)
		c.Writer.Header().Set(RequestIDHeader, rid)

		c.Next()
	}
}

// 추후 정책 변경 시 해당 코드만 바꿔서 수정
func RequestID() gin.HandlerFunc {
	return RequestIDStrict()
}

// 컨텍스트의 리퀘스트 아이디 가져오기
func GetRequestID(c *gin.Context) string {
	v, ok := c.Get(CtxKeyRequestID)
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// 내부 서비스에서 새로 만든 요청에 리퀘스트 아이디 붙일때 사용하는 헬퍼
func WithRequestIDHeader(r *http.Request, rid string) {
	if rid != "" {
		r.Header.Set(RequestIDHeader, rid)
	}
}
