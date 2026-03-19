# Session Policy

## Overview

본 시스템은 JWT와 Redis 기반 세션 모델을 함께 사용합니다.

- JWT: 인증 토큰 역할
- Redis: 세션 상태 관리

---

## Session Definition

세션은 다음 조건을 만족할 때 유효합니다:

- Redis에 해당 사용자 세션이 존재합니다.
- TTL(Time-To-Live) 내에 있습니다.

---

## Session Validation

보호된 API 요청 시 다음 절차를 수행합니다:

1. Gateway에서 인증을 수행합니다.
2. 서비스에서 Redis 세션 존재 여부를 확인합니다.

세션이 존재하지 않을 경우:

→ 401 Unauthorized를 반환합니다.

---

## Session Invalidation

다음 경우 세션이 무효화됩니다:

- Logout 요청 수행 시
- TTL 만료 시
- 관리자에 의한 강제 종료 (향후 확장)

---

## Behavior

- Logout 이후에도 Access Token 자체는 만료 전까지 유효할 수 있습니다.
- 하지만 서버는 Redis 세션 존재 여부를 기준으로 요청을 검증하므로, 세션이 삭제된 이후에는 모든 요청이 거부됩니다.

---

## Rationale

JWT만 사용할 경우 Logout이 즉시 반영되지 않는 문제가 있습니다.

Redis 세션을 통해 해당 문제를 해결합니다.
