# Failure Scenarios

이 문서는 Identity Platform에서 발생할 수 있는 주요 장애 상황과  
각 상황에 대한 시스템 동작을 설명합니다.

Identity Platform은 다음과 같은 장애 상황을 고려하여 설계되었습니다.

---

# Redis Down

## Scenario

Redis가 일시적으로 다운되거나 네트워크 장애로 인해  
Auth 서비스에서 Redis에 접근할 수 없는 상황

## Impact

Redis는 다음 기능에 사용됩니다.

- session validation
- refresh token rotation
- refresh idempotency
- rate limiting

따라서 Redis가 다운되면 인증 검증이 정상적으로 수행되지 않을 수 있습니다.

## Behavior

Redis 접근 실패 시 요청은 **인증 실패로 처리합니다.**

```
HTTP 401 Unauthorized
```

이 방식은 다음 이유로 선택되었습니다.

- 인증 상태를 확실하게 판단할 수 없는 경우 요청을 허용하지 않기 위함
- 보안 측면에서 fail-open 보다 fail-closed 전략이 안전함

---

# Session Deleted

## Scenario

사용자가 logout을 수행하거나  
관리자가 세션을 강제로 삭제한 상황

```
DEL sess:{uid}
```

## Behavior

이후 요청에서 Redis session이 존재하지 않으면  
해당 요청은 인증 실패로 처리됩니다.

```
HTTP 401 Unauthorized
```

이 방식으로 **logout 즉시 반영**이 가능합니다.

---

# Refresh Token Reuse

## Scenario

이미 사용된 refresh token이 다시 사용되는 경우

예:

1️⃣ refresh token 탈취  
2️⃣ 공격자가 이전 refresh token 재사용

## Detection

Refresh 요청 시 다음 검증이 수행됩니다.

```
GET rjti:{uid}
```

토큰의 JTI와 Redis 값이 일치하지 않으면  
**reuse 공격으로 판단합니다.**

## Behavior

요청을 거부합니다.

```
HTTP 401 Unauthorized
```

이 구조를 통해 refresh replay 공격을 방지합니다.

---

# Concurrent Refresh Requests

## Scenario

클라이언트가 refresh 요청을 동시에 여러 번 보내는 경우.

예:

- 네트워크 재시도
- 더블 클릭
- 모바일 재전송

## Risk

동시에 refresh가 수행되면 다음 문제가 발생할 수 있습니다.

- refresh token rotation 꼬임
- 여러 개의 refresh token 발급

## Protection

Redis lock을 사용하여 refresh 요청을 직렬화합니다.

```
SET idem:refresh:{uid} NX EX <ttl>
```

## Behavior

| 상황      | 결과         |
| --------- | ------------ |
| lock 성공 | refresh 처리 |
| lock 실패 | 요청 거부    |

---

# Login Brute Force

## Scenario

공격자가 로그인 endpoint에 대해  
대량의 로그인 시도를 수행하는 경우.

## Protection

IP 기반 rate limit 적용

```
rate:login:{ip}
```

예:

```
5 requests / minute
```

## Behavior

요청 제한 초과 시

```
HTTP 429 Too Many Requests
```

---

# Refresh Abuse

## Scenario

클라이언트가 refresh endpoint를 과도하게 호출하는 경우.

예:

- 잘못된 retry 로직
- refresh abuse 공격

## Protection

사용자 기준 rate limit 적용

```
rate:refresh:{uid}
```

예:

```
10 requests / minute
```

## Behavior

요청 제한 초과 시

```
HTTP 429 Too Many Requests
```

---

# Gateway Bypass Attempt

## Scenario

클라이언트가 Gateway를 우회하여  
Auth 서비스에 직접 요청을 보내는 경우.

## Protection

Auth 서비스는 다음 헤더를 검증합니다.

```
X-Gateway-Verified
```

Gateway에서만 해당 헤더를 추가합니다.

## Behavior

헤더가 존재하지 않으면 요청을 거부합니다.

```
HTTP 403 Forbidden
```

이 구조를 통해 **Gateway boundary enforcement**를 보장합니다.

---

# Summary

Identity Platform은 다음 보안 문제를 고려하여 설계되었습니다.

| Problem            | Protection                |
| ------------------ | ------------------------- |
| JWT logout 문제    | Redis session             |
| refresh replay     | refresh rotation          |
| concurrent refresh | idempotency lock          |
| login brute force  | login rate limit          |
| refresh abuse      | refresh rate limit        |
| gateway bypass     | gateway header validation |

이러한 설계를 통해 인증 시스템의 안정성과 보안을 강화합니다.
