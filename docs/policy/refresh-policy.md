# Refresh Token Policy

## Overview

Refresh Token은 Access Token을 재발급받기 위한 장기 token입니다.

다만 Identity Platform에서는 Refresh Token만으로 Access Token을 재발급하지 않습니다. Refresh 요청은 Refresh Token 검증뿐 아니라 Redis Session 존재 여부도 함께 확인합니다.

현재 기본 TTL 정책은 다음과 같습니다.

```env
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
SESSION_TTL=168h
```

Refresh Token과 Redis Session은 동일한 기본 TTL을 사용합니다. 이를 통해 사용자가 재로그인 없이 유지할 수 있는 기간을 7일로 일관되게 관리합니다.

---

## Validation Steps

Refresh 요청은 다음 검증을 수행합니다.

1. Refresh Token의 서명을 검증합니다.
2. Refresh Token의 만료 여부를 확인합니다.
3. Redis Session이 존재하는지 확인합니다.
4. Refresh 요청 rate limit을 확인합니다.
5. Redis에 저장된 현재 Refresh Token JTI를 조회합니다.
6. 요청 Refresh Token의 JTI와 Redis에 저장된 JTI가 일치하는지 확인합니다.
7. JTI 기준 idempotency lock을 획득합니다.

위 조건 중 하나라도 실패하면 Access Token은 재발급되지 않습니다.

---

## Relationship With Session

Refresh Token은 단독으로 Access Token 재발급 권한을 가지지 않습니다.

Redis Session은 active login marker입니다. Refresh 요청 시 Session을 확인하는 이유는 logout 또는 session 삭제 이후에도 Refresh Token이 남아 있을 수 있기 때문입니다.

따라서 다음 정책이 적용됩니다.

- Session이 살아 있고 Refresh Token도 유효하면 Access Token 재발급이 가능합니다.
- Session이 없으면 Refresh Token이 아직 만료되지 않았더라도 재발급은 실패합니다.
- Logout 시 Session과 Refresh JTI를 삭제하므로 이후 refresh 요청은 실패합니다.

---

## Rotation

Refresh Token은 1회 사용 후 rotation됩니다.

- refresh 성공 시 새로운 Access Token을 발급합니다.
- refresh 성공 시 새로운 Refresh Token을 발급합니다.
- Redis에 저장된 현재 Refresh Token JTI를 새 JTI로 갱신합니다.
- 이전 Refresh Token은 Redis JTI 불일치로 재사용할 수 없습니다.

---

## Idempotency

중복 요청 방지를 위해 JTI 기반 lock을 사용합니다.

현재 코드 기준 lock key는 다음 형태입니다.

```text
idem:refresh:{jti}
```

이 lock은 사용자 전체 단위가 아니라 동일 Refresh Token으로 들어오는 중복/동시 요청을 제어하기 위한 정책입니다.

---

## Error Handling

| 상황 | 응답 |
| --- | --- |
| JSON binding 실패 | `400 Bad Request` |
| Refresh Token 누락 | `400 Bad Request` |
| Token 검증 실패 | `401 Unauthorized` |
| Session 없음 | `401 Unauthorized` |
| Redis JTI 없음 또는 JTI 불일치 | `401 Unauthorized` |
| Rate limit 초과 | `429 Too Many Requests` |
| Redis 내부 error | `500 Internal Server Error` |

---

## Rationale

Refresh Token은 공격 표면이 큰 장기 token입니다. 따라서 Refresh Token 자체의 검증만으로 Access Token을 재발급하지 않고, Redis Session과 Refresh JTI 상태를 함께 확인합니다.

이 정책을 통해 다음을 보장합니다.

- logout 이후 refresh 차단
- session 삭제 이후 refresh 차단
- refresh token replay 차단
- 동시 refresh 요청 제어
