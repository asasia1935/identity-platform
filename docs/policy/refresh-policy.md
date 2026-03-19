# Refresh Token Policy

## Overview

Refresh Token은 Access Token을 재발급하기 위한 토큰입니다.

---

## Validation Steps

Refresh 요청 시 다음 검증을 수행합니다:

1. Refresh Token의 서명 검증
2. 토큰 만료 여부 확인
3. Redis에 저장된 JTI 존재 여부 확인
4. 세션 존재 여부 확인

위 조건 중 하나라도 실패할 경우:

→ 401 Unauthorized를 반환합니다.

---

## Rotation

Refresh Token은 1회 사용 원칙을 따릅니다.

- 새로운 Refresh Token을 발급합니다.
- 기존 Token은 즉시 무효화됩니다.

---

## Idempotency

중복 요청 방지를 위해 JTI 기반 Lock을 사용할 수 있습니다.

---

## Behavior

- Logout 이후 Refresh 요청은 실패해야 합니다.
- 즉, 세션이 존재하지 않을 경우 Refresh는 허용되지 않습니다.

---

## Error Handling

| 상황             | 응답 |
| ---------------- | ---- |
| JSON 바인딩 실패 | 400  |
| Token 검증 실패  | 401  |
| 세션 없음        | 401  |

---

## Rationale

Refresh Token은 공격 표면이 될 수 있으므로,
엄격한 상태 기반 검증이 필요합니다.
