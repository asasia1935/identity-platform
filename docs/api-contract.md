# API Contract

## 1. 문서 목적

이 문서는 Identity Platform의 현재 API 계약을 정리하기 위한 문서입니다.
Gateway, Auth Service, Redis 기반 세션 저장소, 그리고 향후 WalkQuest 같은 downstream service가 어떤 경계로 연결되는지 설명합니다.

본 문서는 현재 코드와 기존 README/docs를 기준으로 작성되었습니다. 코드에서 확인되지 않았거나 정책 결정이 필요한 항목은 "확인 필요" 또는 "추후 보강"으로 표시합니다.

Identity Platform의 핵심 경계는 다음과 같습니다.

- Client는 Auth Service를 직접 호출하지 않고 Gateway를 통해 호출합니다.
- Gateway는 보호 API 요청에서 access token을 검증하고 사용자 컨텍스트를 downstream service로 전달합니다.
- Auth Service와 downstream service는 Gateway가 주입한 인증 컨텍스트를 신뢰합니다.
- Auth Service는 Redis session, refresh JTI, refresh idempotency lock, rate limit 상태를 사용합니다.

## 2. 전체 API 접근 구조

Client가 사용하는 외부 API 경로와 Auth Service 내부 경로는 구분됩니다.

| 구분               | 경로 예시              | 설명                                                                         |
| ------------------ | ---------------------- | ---------------------------------------------------------------------------- |
| Gateway path       | `POST /api/auth/login` | 외부 client가 호출하는 경로입니다.                                           |
| Auth internal path | `POST /auth/login`     | Gateway가 `/api` prefix를 제거한 뒤 Auth Service로 전달하는 내부 경로입니다. |

`cmd/gateway/main.go` 기준으로 Gateway는 다음 경로를 제공합니다.

- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `GET /api/auth/me`
- `POST /api/auth/logout`

`internal/gateway/proxy.go` 기준으로 Gateway reverse proxy는 `/api` prefix를 제거하여 Auth Service에 전달합니다. 예를 들어 `GET /api/auth/me`는 Auth Service의 `GET /auth/me`로 전달됩니다.

Docker Compose 기준으로 application 외부에 공개되는 API 진입점은 Gateway입니다.

- Gateway: `18090:8090`으로 외부 포트가 열립니다.
- Auth: `expose: 8080`으로 compose 내부 네트워크에서만 접근하는 전제입니다.
- Redis: 로컬 개발 편의를 위해 `6379:6379` 포트가 열려 있습니다. 이는 application API 진입점이 아니라 로컬 Redis 확인 목적입니다. 운영 환경에서는 Redis 포트를 외부에 공개하지 않는 것을 전제로 합니다.

## 3. 공통 요청 규칙

JSON 요청 body를 보내는 API는 다음 header를 사용합니다.

```http
Content-Type: application/json
```

보호 API는 다음 access token header가 필요합니다.

```http
Authorization: Bearer <access_token>
```

Gateway는 보호 API에서 access token 검증에 성공하면 downstream 요청에 사용자 컨텍스트를 주입합니다.

```http
X-Gateway-Verified: true
X-User-ID: <authenticated_user_id>
```

`X-Gateway-Verified`는 Gateway reverse proxy가 Auth Service로 전달할 때 설정합니다. `X-User-ID`는 보호 API에서 Gateway 인증 미들웨어가 access token의 subject를 추출해 설정합니다.

외부 client가 Auth Service 또는 downstream service에 직접 `X-Gateway-Verified`, `X-User-ID`를 넣어 호출하면 안 됩니다. 이 header들은 client 입력값이 아니라 Gateway가 인증 후 주입하는 내부 계약입니다.

Downstream service는 Gateway가 주입한 header만 신뢰해야 합니다. 운영 환경에서는 이 신뢰 모델이 성립하도록 서비스 포트 노출, ingress, firewall/security group, private network, mTLS 또는 service mesh 같은 네트워크 경계 보강이 필요할 수 있습니다.

## 4. 공통 응답 규칙

성공 응답은 endpoint별 JSON body 또는 empty body를 사용합니다.

- token 발급/재발급: JSON body
- `GET /auth/me`: JSON body
- `POST /auth/logout`: `204 No Content`, body 없음

에러 응답은 현재 코드 기준으로 다음 형태의 JSON body를 사용합니다.

```json
{
  "error": "error_code"
}
```

현재 코드에서 확인되는 error code 예시는 다음과 같습니다.

| Error code             | 사용 경로 예시                                      |
| ---------------------- | --------------------------------------------------- |
| `invalid json`         | login JSON binding 실패                             |
| `bad_request`          | refresh JSON binding 실패 또는 refresh token 누락   |
| `invalid credentials`  | login credential 실패                               |
| `unauthorized`         | token 검증 실패, session 없음, refresh 검증 실패 등 |
| `missing user context` | Auth 내부 보호 API에서 `X-User-ID` 없음             |
| `gateway required`     | Auth 내부 API에서 `X-Gateway-Verified: true` 없음   |
| `too_many_requests`    | login 또는 refresh rate limit 초과                  |
| `internal`             | Redis error 등 Auth 내부 처리 실패                  |
| `upstream unavailable` | Gateway upstream 연결 실패                          |
| `upstream timeout`     | Gateway upstream timeout                            |

Status code 정책은 `docs/policy/error-policy.md`, `docs/failure-scenarios.md`, 현재 handler 구현을 함께 기준으로 정리합니다. 세부 내용은 7장을 참고하십시오.

## 5. Auth API 명세

### 5.1 Login

| 항목               | 내용                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------ |
| Gateway path       | `POST /api/auth/login`                                                               |
| Auth internal path | `POST /auth/login`                                                                   |
| 인증 필요 여부     | access token 인증 요구 X. 단, Auth internal path는 Gateway 경유 header가 필요합니다. |

Request headers:

```http
Content-Type: application/json
```

Gateway가 Auth Service로 전달할 때 내부적으로 다음 header를 주입합니다.

```http
X-Gateway-Verified: true
```

Request body:

```json
{
  "username": "test",
  "password": "1234"
}
```

현재 코드 기준 credential은 `username=test`, `password=1234`로 하드코딩되어 있습니다. 실제 사용자 저장소 연동은 추후 보강 대상입니다.

Success response:

```http
HTTP/1.1 200 OK
```

```json
{
  "access_token": "...",
  "refresh_token": "..."
}
```

성공 시 Auth Service는 다음 작업을 수행합니다.

- access token 발급
- refresh token 발급
- Redis session 생성
- Redis refresh JTI 저장

Error cases:

| 조건                                               | Status                      | Body 예시                                                    |
| -------------------------------------------------- | --------------------------- | ------------------------------------------------------------ |
| JSON body 파싱 실패                                | `400 Bad Request`           | `{"error":"invalid json"}`                                   |
| credential 불일치                                  | `401 Unauthorized`          | `{"error":"invalid credentials"}`                            |
| login rate limit 초과                              | `429 Too Many Requests`     | `{"error":"too_many_requests"}`                              |
| rate limit Redis error                             | `500 Internal Server Error` | `{"error":"internal"}`                                       |
| token 발급 실패                                    | `500 Internal Server Error` | `{"error":"token issue failed"}` 또는 `{"error":"internal"}` |
| session 저장 실패                                  | `500 Internal Server Error` | `{"error":"internal"}`                                       |
| refresh JTI 저장 실패                              | `500 Internal Server Error` | `{"error":"internal"}`                                       |
| Auth internal path를 Gateway header 없이 직접 호출 | `403 Forbidden`             | `{"error":"gateway required"}`                               |

### 5.2 Refresh

| 항목               | 내용                                                                                                 |
| ------------------ | ---------------------------------------------------------------------------------------------------- |
| Gateway path       | `POST /api/auth/refresh`                                                                             |
| Auth internal path | `POST /auth/refresh`                                                                                 |
| 인증 필요 여부     | access token 인증 요구 X. refresh token 필요. Auth internal path는 Gateway 경유 header가 필요합니다. |

Request headers:

```http
Content-Type: application/json
```

Gateway가 Auth Service로 전달할 때 내부적으로 다음 header를 주입합니다.

```http
X-Gateway-Verified: true
```

Request body:

```json
{
  "refresh_token": "..."
}
```

Success response:

```http
HTTP/1.1 200 OK
```

```json
{
  "access_token": "...",
  "refresh_token": "..."
}
```

현재 코드와 정책 문서 기준 refresh 요청은 다음 검증을 거칩니다.

1. refresh token 서명과 만료 여부를 검증합니다.
2. refresh token claims에서 user id와 JTI를 추출합니다.
3. Redis session 존재 여부를 확인합니다.
4. refresh rate limit을 사용자 기준으로 확인합니다.
5. Redis에 저장된 현재 refresh JTI를 조회합니다.
6. 요청 refresh token의 JTI와 Redis JTI가 일치하는지 확인합니다.
7. refresh idempotency lock 획득을 시도합니다.
8. 새 access token과 새 refresh token을 발급합니다.
9. Redis refresh JTI를 새 값으로 갱신합니다.

Session이 없으면 refresh는 실패할 수 있습니다. 이는 현재 session policy와 refresh policy에 명시된 동작이며, logout 이후 refresh를 차단하기 위한 구조입니다.

Idempotency/retry 방지 정책:

- 현재 코드 기준 lock key는 refresh token의 `JTI`를 사용하며, key 형태는 `idem:refresh:{jti}`입니다.
- 이 lock은 사용자 전체 단위가 아니라 동일 refresh token으로 들어오는 중복/동시 요청을 제어하기 위한 목적입니다.

Error cases:

| 조건                                               | Status                      | Body 예시                       |
| -------------------------------------------------- | --------------------------- | ------------------------------- |
| JSON body 파싱 실패                                | `400 Bad Request`           | `{"error":"bad_request"}`       |
| `refresh_token` 누락 또는 빈 문자열                | `400 Bad Request`           | `{"error":"bad_request"}`       |
| refresh token 검증 실패 또는 만료                  | `401 Unauthorized`          | `{"error":"unauthorized"}`      |
| Redis session 없음                                 | `401 Unauthorized`          | `{"error":"unauthorized"}`      |
| Redis refresh JTI 없음                             | `401 Unauthorized`          | `{"error":"unauthorized"}`      |
| refresh token JTI와 Redis JTI 불일치               | `401 Unauthorized`          | `{"error":"unauthorized"}`      |
| idempotency lock 획득 실패                         | `401 Unauthorized`          | `{"error":"unauthorized"}`      |
| refresh rate limit 초과                            | `429 Too Many Requests`     | `{"error":"too_many_requests"}` |
| Redis 내부 error                                   | `500 Internal Server Error` | `{"error":"internal"}`          |
| Auth internal path를 Gateway header 없이 직접 호출 | `403 Forbidden`             | `{"error":"gateway required"}`  |

### 5.3 Logout

| 항목               | 내용                    |
| ------------------ | ----------------------- |
| Gateway path       | `POST /api/auth/logout` |
| Auth internal path | `POST /auth/logout`     |
| 인증 필요 여부     | access token 필요       |

Request headers:

```http
Authorization: Bearer <access_token>
```

Gateway가 인증 성공 후 Auth Service로 전달할 때 다음 header를 주입합니다.

```http
X-Gateway-Verified: true
X-User-ID: <authenticated_user_id>
```

Request body:

없습니다.

Success response:

```http
HTTP/1.1 204 No Content
```

Logout 성공 시 Auth Service는 다음 작업을 수행합니다.

- Redis session 삭제
- Redis refresh JTI 삭제

현재 코드 기준 session 삭제 후 refresh JTI 삭제가 실패하면 `500 Internal Server Error`를 반환합니다. 다만 session 삭제가 이미 성공한 경우 logout을 성공으로 볼지, refresh JTI 삭제 실패까지 엄격하게 실패로 볼지는 정책 확정이 필요합니다.

Error cases:

| 조건                                               | Status                      | Body 예시                          |
| -------------------------------------------------- | --------------------------- | ---------------------------------- |
| access token 누락, 형식 오류, 검증 실패            | `401 Unauthorized`          | `{"error":"unauthorized"}`         |
| Auth 내부 요청에서 `X-User-ID` 없음                | `401 Unauthorized`          | `{"error":"missing user context"}` |
| session 삭제 Redis error                           | `500 Internal Server Error` | `{"error":"internal"}`             |
| refresh JTI 삭제 Redis error                       | `500 Internal Server Error` | `{"error":"internal"}`             |
| Auth internal path를 Gateway header 없이 직접 호출 | `403 Forbidden`             | `{"error":"gateway required"}`     |

### 5.4 Me

| 항목               | 내용               |
| ------------------ | ------------------ |
| Gateway path       | `GET /api/auth/me` |
| Auth internal path | `GET /auth/me`     |
| 인증 필요 여부     | access token 필요  |

Request headers:

```http
Authorization: Bearer <access_token>
```

Gateway가 인증 성공 후 Auth Service로 전달할 때 다음 header를 주입합니다.

```http
X-Gateway-Verified: true
X-User-ID: <authenticated_user_id>
```

Success response:

```http
HTTP/1.1 200 OK
```

```json
{
  "user": "test"
}
```

현재 코드 기준 response key는 `internal/mw/headers.go`의 `ContextUserKey` 값인 `user`입니다.

Error cases:

| 조건                                               | Status                      | Body 예시                          |
| -------------------------------------------------- | --------------------------- | ---------------------------------- |
| access token 누락, 형식 오류, 검증 실패            | `401 Unauthorized`          | `{"error":"unauthorized"}`         |
| Auth 내부 요청에서 `X-User-ID` 없음                | `401 Unauthorized`          | `{"error":"missing user context"}` |
| Redis session 없음                                 | `401 Unauthorized`          | `{"error":"unauthorized"}`         |
| Redis session check error                          | `500 Internal Server Error` | `{"error":"internal"}`             |
| Auth internal path를 Gateway header 없이 직접 호출 | `403 Forbidden`             | `{"error":"gateway required"}`     |

## 6. Token Response Field Policy

Token 응답 필드는 snake_case로 통일합니다.

```json
{
  "access_token": "...",
  "refresh_token": "..."
}
```

`accessToken`, `refreshToken` 같은 camelCase token 응답 필드는 사용하지 않습니다.

이 정책은 WalkQuest 또는 다른 client가 login과 refresh 응답을 같은 방식으로 파싱하도록 하기 위한 계약입니다. Endpoint마다 field naming이 달라지는 경우 client SDK, Postman collection, frontend integration, automated test에서 혼선이 생길 수 있으므로 token 응답 field는 `access_token`, `refresh_token`만 사용합니다.

## 7. Status Code Policy

현재 코드와 기존 정책 문서 기준 status code는 다음과 같이 정리합니다.

| Status                      | 의미                          | 현재 코드/문서 기준                                       |
| --------------------------- | ----------------------------- | --------------------------------------------------------- |
| `200 OK`                    | 요청 성공                     | login, refresh, me 성공                                   |
| `204 No Content`            | body 없는 성공                | logout 성공                                               |
| `400 Bad Request`           | 잘못된 요청                   | JSON binding 실패, refresh token 누락                     |
| `401 Unauthorized`          | 인증 실패 또는 인증 상태 무효 | invalid/expired token, session missing, refresh 검증 실패 |
| `403 Forbidden`             | Gateway 경계 위반             | Auth internal path에 `X-Gateway-Verified: true` 없음      |
| `429 Too Many Requests`     | rate limit 초과               | login/refresh rate limit 초과                             |
| `500 Internal Server Error` | Auth 내부 처리 실패           | Redis error, token 발급 실패 등                           |
| `502 Bad Gateway`           | Gateway upstream 연결 실패    | Auth service unavailable 등                               |
| `504 Gateway Timeout`       | Gateway upstream timeout      | Auth 응답 지연, upstream timeout                          |

502와 504는 Auth Service handler가 직접 반환하는 상태라기보다 Gateway가 upstream 호출 실패/timeout을 감지했을 때 client에게 반환하는 상태입니다.

명확히 구분해야 하는 정책은 다음과 같습니다.

- session missing 또는 Redis에 세션 없음: `401 Unauthorized`
- invalid/expired access token: `401 Unauthorized`
- invalid/expired refresh token: `401 Unauthorized`
- gateway bypass 또는 gateway verified header 없음: `403 Forbidden`
- rate limit exceeded: `429 Too Many Requests`
- Auth 내부 Redis error: `500 Internal Server Error`
- Gateway upstream connection failure: `502 Bad Gateway`
- Gateway upstream timeout: `504 Gateway Timeout`

확인 필요:

- Redis 장애가 Auth 내부에서 즉시 error로 반환되면 `500`이지만, Gateway timeout까지 지연되면 client는 `504`를 받을 수 있습니다. 운영 timeout 기준과 Redis client timeout 정책은 추후 보강이 필요합니다.
- Concurrent refresh에서 idempotency lock 획득 실패를 현재 코드는 `401 Unauthorized`로 처리합니다. 이 status가 최종 정책인지 확인이 필요합니다.
- Logout 중 session 삭제 성공 후 refresh JTI 삭제 실패를 `500`으로 유지할지, idempotent logout으로 간주해 `204`로 볼지 확인이 필요합니다.

## 8. Gateway / Downstream Service Contract

향후 WalkQuest 같은 downstream service가 붙을 때의 최소 계약은 다음과 같습니다.

- Protected downstream API는 Gateway를 통해서만 접근합니다.
- Gateway는 protected API 요청의 access token을 검증합니다.
- Gateway는 인증된 사용자 ID를 `X-User-ID`로 주입합니다.
- Gateway는 downstream 요청에 `X-Gateway-Verified: true`를 주입합니다.
- Downstream service는 JWT를 직접 검증하지 않습니다.
- Downstream service는 `X-User-ID`를 authenticated user context로 사용합니다.
- Downstream service 포트는 외부에 직접 노출하지 않습니다.
- 로컬 개발 환경에서는 Docker Compose network를 통해 Gateway-only 구조를 유지합니다.
- 운영 환경에서는 private network, ingress rule, firewall/security group, mTLS 또는 service mesh로 Gateway trust boundary를 보강할 수 있습니다.

Downstream service는 client가 임의로 보낸 `X-User-ID`를 신뢰하면 안 됩니다. 해당 header는 Gateway에서 인증 성공 후 주입된 값이라는 네트워크 경계 전제가 있어야 합니다.

WalkQuest 연동 시에는 다음 contract test를 추가하는 것이 좋습니다.

- Client가 Gateway protected path로 access token을 보내면 downstream에 `X-User-ID`가 전달되는지 확인합니다.
- Client가 임의로 보낸 `X-User-ID`가 Gateway에서 인증된 사용자 ID로 덮어써지는지 확인합니다.
- Downstream service를 직접 호출하는 경로가 외부에서 차단되는지 확인합니다.
- 세션 없음, token 만료, 권한 부족, rate limit, upstream timeout의 status code가 계약과 일치하는지 확인합니다.

## 9. Local Development Notes

Docker Compose 기준 local 접근 방식은 다음과 같습니다.

- Gateway 외부 접근: `http://localhost:18090`
- Auth service: compose 내부에서 `http://auth:8080`으로 접근합니다.
- Gateway의 `AUTH_UPSTREAM`은 compose 기준 `http://auth:8080`입니다.
- Auth container는 `expose: 8080`만 사용하므로 application client는 Gateway path를 사용해야 합니다.
- Redis는 로컬 확인 목적으로 `6379:6379`이 열려 있습니다.

curl 또는 Postman 테스트 시에는 Gateway 경로를 사용해야 합니다.

예시:

```bash
curl -X POST http://localhost:18090/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"1234"}'
```

보호 API 예시:

```bash
curl http://localhost:18090/api/auth/me \
  -H "Authorization: Bearer <access_token>"
```

직접 Auth Service를 호출하는 경우는 내부 테스트/개발 목적이어야 합니다. Auth internal path는 `X-Gateway-Verified: true`와 보호 API의 경우 `X-User-ID`가 필요하므로 일반 client 호출 경로로 사용하지 않습니다.

`.env.example` 기준으로 binary를 직접 실행하는 로컬 환경에서는 Gateway 기본 port가 `8090`, Auth upstream 기본값이 `http://localhost:8080`입니다. Docker Compose와 직접 실행 환경의 port가 다르므로 테스트 시 실행 방식을 먼저 확인해야 합니다.

## 10. 남은 확인 사항

추후 확인/보강할 항목은 다음과 같습니다.

- Session TTL과 refresh TTL 정책 확정
  - 현재 `.env.example` 기준 `SESSION_TTL=15m`, `REFRESH_TOKEN_TTL=168h`입니다.
  - Refresh 요청은 session 존재 여부를 확인하므로 session TTL이 짧으면 refresh token이 남아 있어도 refresh가 실패할 수 있습니다.
- Logout 부분 실패 정책 확정
  - session 삭제 성공 후 refresh JTI 삭제 실패 시 `204`로 볼지, 현재처럼 `500`으로 볼지 결정이 필요합니다.
- WalkQuest 연동 후 downstream contract 테스트 추가
  - Gateway header 주입, direct access 차단, status code 계약을 통합 테스트로 확인해야 합니다.
- README와 API contract 문서 간 링크 정리
  - README의 Documentation 섹션에서 본 문서로 연결하는 작업은 별도 작업으로 진행할 수 있습니다.
