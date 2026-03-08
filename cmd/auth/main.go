package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/asasia1935/identity-platform/internal/config"
	"github.com/asasia1935/identity-platform/internal/mw"
	"github.com/asasia1935/identity-platform/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func NewRouter(tm *auth.TokenManager, ss auth.SessionStore, rs auth.RefreshStore) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(mw.RequestID())
	r.Use(gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		rid := p.Request.Header.Get(mw.RequestIDHeader) // 헤더에서 꺼내기
		// gin.Context Keys에서 꺼내려면 Custom middleware가 필요함(Formatter는 gin.Context를 직접 못 받음)

		// 상태코드가 400 이상일때 플래그 WARN으로 변경
		level := "INFO"
		if p.StatusCode >= 400 {
			level = "WARN"
		}

		// 경로 로그 추가
		path := p.Path
		if path == "" {
			path = p.Request.URL.Path
		}

		return fmt.Sprintf(
			"level=%s svc=%s rid=%s method=%s path=%s status=%d latency_ms=%d ip=%s\n",
			level, "gateway", rid, p.Method, path, p.StatusCode, p.Latency.Milliseconds(), p.ClientIP,
		)
	}))

	// 504 타임아웃 테스트용 느린 서버
	r.GET("/auth/slow", func(c *gin.Context) {
		time.Sleep(10 * time.Second)
		c.String(200, "ok")
	})

	// Auth Health Check는 Gateway 체크 예외
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// 아래 API는 게이트웨이를 필수로 거쳐야함
	r.Use(mw.GatewayRequired())

	r.GET("/me", mw.JWTRequired(tm), func(c *gin.Context) {

		user, _ := c.Get("user")

		ok, err := ss.Exists(c.Request.Context(), user.(string))
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": "internal"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": user,
		})
	})

	r.POST("/auth/login", func(c *gin.Context) {
		// 로그인 요청 바디(JSON) 구조 정의
		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		var req LoginRequest

		// JSON 바인딩
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid json",
			})
			return
		}

		// 임시로 인증 (하드코딩)
		if req.Username != "test" || req.Password != "1234" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid credentials",
			})
			return
		}

		// Access Token 생성
		access, err := tm.GenerateAccessToken(req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "token issue failed",
			})
			return
		}

		// Refresh Token은 JTI도 같이 반환 (클레임에 넣어서 반환)
		refresh, jti, err := tm.GenerateRefreshToken(req.Username)
		if err != nil {
			c.JSON(500, gin.H{"error": "internal"})
			return
		}

		// 세션 추가
		if err := ss.Create(c.Request.Context(), req.Username); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// refresh jti 저장
		if err := rs.Save(c.Request.Context(), req.Username, jti); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// Access/Refresh Token 모두 반환
		c.JSON(http.StatusOK, gin.H{
			"accessToken":  access,
			"refreshToken": refresh,
		})
	})

	// POST /auth/logout
	r.POST("/auth/logout", mw.JWTRequired(tm), func(c *gin.Context) {
		v, ok := c.Get("user")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		uid, ok := v.(string)
		if !ok || uid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// 세션 삭제
		if err := ss.Delete(c.Request.Context(), uid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// 로그아웃은 응답 데이터가 없기 때문에 204로 전달
		c.Status(http.StatusNoContent)
	})

	// POST /auth/refresh -> Access Token 재발급 API
	r.POST("/auth/refresh", func(c *gin.Context) {
		// Refresh 요청 바디 구조 정의
		type RefreshRequest struct {
			RefreshToken string `json:"refresh_token"`
		}

		// Refresh 요청 바디 바인딩
		var req RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
			return
		}

		// Refresh Token 검증 (클레임까지 반환)
		claims, err := tm.VerifyRefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Refresh Token에서 UID와 JTI 추출
		uid := claims.Subject
		// JTI는 Refresh Token의 고유 식별자 (claims에서 추출)

		// Redis에서 현재 유효한 Refresh Token의 JTI 조회
		currentJTI, err := rs.Get(c.Request.Context(), uid)
		if err != nil {
			// Redis에 키가 없는 경우 (즉, 유효한 Refresh Token이 없는 경우) -> 401 Unauthorized
			if errors.Is(err, redis.Nil) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			// 그 외 Redis 오류는 서버 오류로 처리
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// Redis에서 조회한 JTI와 Refresh Token의 JTI가 일치하는지 검증 (불일치하면 401 Unauthorized)
		if currentJTI != claims.JTI {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Access Token 재발급
		accessToken, err := tm.GenerateAccessToken(uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// Access Token만 재발급해서 반환 (Refresh Token은 그대로 유지 -> Rotation은 추후에 구현)
		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
		})
	})

	return r
}

func main() {
	_ = godotenv.Load() // .env 없으면 무시

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	rdb, err := store.NewRedisClient(store.RedisConfig{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	tm, err := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		log.Fatal(err)
	}

	ss, err := auth.NewRedisSessionStore(rdb, cfg.SessionTTL)
	if err != nil {
		log.Fatal(err)
	}

	rs, err := auth.NewRedisRefreshStore(rdb, cfg.RefreshTokenTTL)
	if err != nil {
		log.Fatal(err)
	}

	r := NewRouter(tm, ss, rs)
	log.Fatal(r.Run(":" + cfg.HTTPPort))
}
