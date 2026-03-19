# Failure Scenarios

이 문서는 Identity Platform에서 발생할 수 있는 주요 장애 및 실패 상황과
각 상황에 대한 시스템 동작을 설명합니다.

Identity Platform은 다음 계층의 실패를 고려하여 설계되었습니다.

- Gateway / Upstream failure
- Redis dependency failure
- Authentication / token validation failure
- Abuse protection failure

---

# Gateway / Upstream Failure

## Auth Service Unavailable

### Scenario

Gateway가 Auth 서비스에 연결할 수 없는 상황입니다.

예:

- Auth 프로세스 비정상 종료
- 잘못된 upstream 주소
- 네트워크 연결 실패

### Behavior

Gateway는 다음 응답을 반환합니다.

```
HTTP 502 Bad Gateway
```

### Rationale

요청은 Gateway까지 정상 도달했지만
upstream 서비스에 전달할 수 없으므로
boundary failure로 처리합니다.

## Upstream Timeout

### Scenario

Gateway가 Auth 서비스로 요청을 전달했지만
설정된 timeout 내에 응답을 받지 못한 상황입니다.

예:

- Auth 내부 처리 지연
- Redis dependency 지연
- 의도적으로 만든 느린 테스트 API 호출

### Behavior

Gateway는 요청을 중단하고 다음 응답을 반환합니다.

```
HTTP 504 Gateway Timeout
```

Rationale

Gateway는 클라이언트 요청이 무기한 대기하지 않도록
upstream timeout을 적용합니다.

이를 통해 장애 상황에서 응답 지연을 제한하고
실패를 명확하게 드러낼 수 있습니다.

---

# Redis / Authentication Failure

## Redis Unavailable

### Scenario

Redis가 다운되거나 네트워크 장애로 인해
Auth 서비스가 Redis에 접근할 수 없는 상황입니다.

### Impact

Redis는 다음 기능에 사용됩니다.

- session validation
- refresh token rotation
- refresh idempotency
- rate limiting

따라서 Redis 장애는 인증 처리 전반에 영향을 미칩니다.

### Observed Behavior

현재 구성에서는 Redis 장애가 발생하면
Auth 서비스 내부에서 Redis 관련 처리가 지연되거나 실패하고,
Gateway 관점에서는 upstream timeout으로 관찰될 수 있습니다.

실제 테스트에서는 다음 응답이 관찰되었습니다.

```
HTTP 504 Gateway Timeout
```

상황에 따라 upstream 연결 실패로 전파되면
다음과 같이 관찰될 수도 있습니다.

```
HTTP 502 Bad Gateway
```

### Implementation Note

코드 상 일부 경로에서는 Redis 접근 실패를
인증 실패로 처리하려는 의도가 있을 수 있지만,

현재 시스템 구성에서는 Redis 장애가
Gateway upstream failure 또는 timeout 형태로
관찰될 수 있습니다.

### Rationale

Redis는 Auth 서비스의 핵심 의존성이므로
Redis 장애는 단순 인증 실패가 아니라
인증 처리 경로 전체의 인프라 장애로 전파될 수 있습니다.

---

## Invalid or Expired Access Token

### Scenario

보호 API 요청에 사용된 access token이
유효하지 않은 상황입니다.

예:

- access token 만료
- 잘못된 서명
- malformed token
- claims 검증 실패

### Behavior

요청을 거부합니다.

```
HTTP 401 Unauthorized
```

### Rationale

유효하지 않은 access token은
인증된 사용자로 간주할 수 없습니다.

---

## Missing Server-side Session

### Scenario

JWT 자체는 형식상 유효하지만
Redis에 저장된 서버 측 세션이 존재하지 않는 상황입니다.

예:

- logout 수행
- session TTL 만료
- 관리자가 세션 삭제

```
DEL sess:{uid}
```

### Behavior

요청을 거부합니다.

```
HTTP 401 Unauthorized
```

### Rationale

이 구조를 통해

- logout 즉시 반영
- 서버 측 세션 강제 종료

를 구현합니다.

---

## Refresh Token Validation Failure

### Scenario

refresh 요청에 사용된 refresh token이
유효하지 않은 상황입니다.

예:

- refresh token 누락
- malformed token
- claims 파싱 실패
- 필요한 claim 누락

Behavior

요청을 거부합니다.

```
HTTP 401 Unauthorized
```

### Rationale

유효하지 않은 refresh token으로는
새 토큰을 발급할 수 없습니다.

---

## Refresh Token Reuse

### Scenario

이미 교체된 이전 refresh token이
다시 사용되는 상황입니다.

예:

- 정상 refresh 요청으로 새 refresh token 발급
- 이전 refresh token이 다시 전송됨
- 탈취 또는 replay 가능성

### Detection

Refresh 요청 시 다음 검증이 수행됩니다.

```
GET rjti:{uid}
```

토큰의 JTI와 Redis 값이 일치하지 않으면
reuse 또는 invalid refresh state로 판단합니다.

Behavior

요청을 거부합니다.

```
HTTP 401 Unauthorized
```

### Rationale

refresh replay 공격을 방지합니다.

---

## Concurrent Refresh Requests

### Scenario

클라이언트가 refresh 요청을
동시에 여러 번 보내는 상황입니다.

예:

- 네트워크 재시도
- 더블 클릭
- 모바일 재전송

### Risk

동시에 refresh가 수행되면

- refresh rotation 꼬임
- 여러 refresh token 발급
- 클라이언트 상태 불일치

문제가 발생할 수 있습니다.

### Protection

Redis lock 사용

```
SET idem:refresh:{uid} 1 NX EX <ttl>
```

### Behavior

|상황       | 결과         |
|-----------|-------------|
| lock 성공 | refresh 처리 |
| lock 실패 | 요청 거부     |

### Rationale

refresh 요청을 직렬화하여
중복 토큰 발급을 방지합니다.

---

## Rate Limit Exceeded

### Scenario

login 또는 refresh endpoint를
과도하게 호출하는 상황입니다.

예:

- login brute force
- refresh abuse
- 잘못된 retry 로직

### Protection

다음 기준으로 rate limit 적용

- login: IP 기준
- refresh: 사용자 기준

```
rate:login:{ip}
rate:refresh:{uid}
```

### Behavior

요청 제한 초과 시

```
HTTP 429 Too Many Requests
```

### Rationale

무차별 로그인 시도와
refresh abuse를 제한합니다.

---

## Gateway Bypass Attempt

### Scenario

클라이언트가 Gateway를 우회하여
Auth 서비스에 직접 요청을 보내는 상황입니다.

### Protection

Auth 서비스는 다음 헤더를 검증합니다.

```
X-Gateway-Verified
```

해당 헤더는 Gateway에서만 추가됩니다.

### Behavior

헤더가 존재하지 않으면 요청을 거부합니다.

```
HTTP 403 Forbidden
```

### Rationale

Auth 서비스가 반드시 Gateway 뒤에서만
동작하도록 강제합니다.

---

Summary

Identity Platform은 다음 실패 상황을 고려하여 설계되었습니다.

|Failure                   |Behavior                       |
|--------------------------|-------------------------------|
|Auth service unavailable  |502 Bad Gateway                |
|Upstream timeout          |504 Gateway Timeout            |
|Redis unavailable         |upstream timeout / failure 가능|
|invalid access token      |401 Unauthorized               |
|missing session           |401 Unauthorized               |
|refresh validation failure|401 Unauthorized               |
|refresh token reuse       |401 Unauthorized               |
|concurrent refresh        |Redis lock                     |
|rate limit exceeded       |429 Too Many Requests          |
|gateway bypass            |403 Forbidden                  |