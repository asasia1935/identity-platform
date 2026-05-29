# Session Policy

## Overview

Identity Platform은 JWT와 Redis 기반 Session을 함께 사용합니다.

- JWT: API 요청 인증에 사용하는 token입니다.
- Redis Session: 사용자의 로그인 상태를 서버 측에서 제어하기 위한 active login marker입니다.

여기서 Session은 전통적인 서버 세션처럼 상세한 사용자 상태를 저장하는 객체가 아닙니다. Redis에 `sess:{uid}` marker가 존재하는지를 기준으로 해당 사용자의 로그인 상태가 active한지 판단합니다.

---

## Session Definition

Session은 다음 조건을 만족할 때 유효합니다.

- Redis에 해당 사용자 session marker가 존재합니다.
- marker의 TTL이 만료되지 않았습니다.

현재 기본 정책은 다음과 같습니다.

```env
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
SESSION_TTL=168h
```

Access Token은 짧게 유지하고, Refresh Token과 Redis Session은 동일한 기본 TTL을 사용하여 로그인 유지 가능 기간을 일관되게 관리합니다.

---

## Session Validation

보호 API 요청은 다음 절차를 따릅니다.

1. Gateway에서 access token을 검증합니다.
2. Gateway가 `X-User-ID`를 Auth Service 또는 downstream service로 전달합니다.
3. Auth Service는 Redis Session 존재 여부를 확인합니다.

Session이 존재하지 않으면 요청은 인증 실패로 처리됩니다.

```http
HTTP/1.1 401 Unauthorized
```

Refresh 요청도 Session 존재 여부를 함께 확인합니다. 즉, Refresh Token이 아직 만료되지 않았더라도 Redis Session이 없으면 Access Token 재발급은 실패합니다.

---

## Session Invalidation

Session은 다음 경우 무효화됩니다.

- Logout 요청 처리 시
- TTL 만료 시
- 관리자 또는 운영 도구에 의한 강제 삭제 시 (추후 확장)

Logout의 핵심 성공 기준은 Redis Session 삭제 성공 여부입니다.
Redis Session은 active login marker이므로, Session 삭제가 성공하면 해당 사용자는 더 이상 로그인 상태로 인정되지 않습니다.

Auth Service는 logout 처리 중 현재 Refresh Token JTI 삭제도 시도합니다.
다만 Session 삭제가 이미 성공한 뒤 Refresh JTI 삭제가 실패하더라도 client에는 `204 No Content`를 반환합니다.
이 경우 사용자는 이미 로그아웃 상태이며, Refresh JTI 삭제 실패는 로그로 남기고 추후 cleanup/retry 대상으로 관리합니다.

Session 삭제 자체가 실패하면 로그인 상태 종료를 보장할 수 없으므로 `500 Internal Server Error`를 반환합니다.

---

## Behavior

- Access Token 만료는 로그아웃이 아닙니다. 클라이언트가 Refresh 요청으로 새 Access Token을 받아야 하는 상태입니다.
- Session 만료 또는 삭제는 로그인 상태 종료에 가깝습니다.
- Session이 살아 있고 Refresh Token도 유효하면, 사용자는 기본 정책상 최대 7일 동안 재로그인 없이 Access Token을 재발급받을 수 있습니다.
- Session이 없으면 protected API 요청과 refresh 요청 모두 실패할 수 있습니다.

---

## Rationale

JWT만 사용할 경우 logout 이후에도 이미 발급된 Access Token이 만료 전까지 남아 있을 수 있습니다. 서버가 이를 즉시 차단하려면 별도의 서버 측 상태가 필요합니다.

Identity Platform은 Redis Session을 active login marker로 사용하여 다음 문제를 해결합니다.

- logout 즉시 반영
- session 삭제를 통한 로그인 상태 종료
- Refresh Token이 남아 있어도 Session이 없으면 Access Token 재발급 차단

이 정책은 JWT의 stateless 장점을 유지하면서도 최소한의 서버 측 상태로 로그인 상태를 제어하기 위한 절충안입니다.
