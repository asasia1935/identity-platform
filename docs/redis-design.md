# Redis Design

이 문서는 Identity Platform에서 사용하는 Redis 데이터 구조와
각 key의 역할을 설명합니다.

Redis는 다음 목적을 위해 사용됩니다:

- Session validation
- Refresh token rotation 관리
- Refresh idempotency 처리
- Rate limiting

---

# Key Overview

| Key | Purpose |
|----|----|
| `sess:{uid}` | 사용자 세션 존재 여부 확인 |
| `refresh:{uid}` | 현재 유효한 refresh token JTI |
| `idem:refresh:{uid}` | refresh 요청 idempotency lock |
| `rate:login:{ip}` | 로그인 rate limit |
| `rate:refresh:{uid}` | refresh rate limit |

---

# Session Key

```
sess:{uid}
```

Example

```
sess:123
```

## Purpose

JWT는 기본적으로 stateless 구조이기 때문에
서버는 토큰이 실제로 유효한 세션인지 확인할 수 없습니다.

이를 보완하기 위해 Redis에 세션을 저장하여
**서버 측 세션 제어를 가능하도록 합니다.**

예:

- Logout 시 세션 삭제
- 세션 강제 만료
- 탈취 대응

## Behavior

Login

```
SET sess:{uid} 1 EX <session_ttl>
```

Protected API

```
GET sess:{uid}
```

Logout

```
DEL sess:{uid}
```

세션 key가 존재하지 않으면 인증 실패로 처리

---

# Refresh Token JTI

```
rjti:{uid}
```

Example

```
rjti:123
```

## Purpose

Refresh token rotation을 구현하기 위해
현재 유효한 refresh token의 JTI를 Redis에 저장합니다.

이 구조를 통해

- refresh replay 공격 방지
- 이전 refresh token 사용 차단

을 구현할 수 있습니다.

## Behavior

Login 시

```
SET rjti:{uid} {jti} EX <refresh_ttl>
```

Refresh 시

1️⃣ 토큰의 JTI와 Redis 값을 비교

```
GET rjti:{uid}
```

2️⃣ 일치하면 새 refresh 발급

3️⃣ Redis 갱신

```
SET rjti:{uid} {new_jti} EX <refresh_ttl>
```

일치하지 않으면 **reuse 공격으로 판단**하고 요청을 거부

---

# Refresh Idempotency Lock

```
idem:refresh:{uid}
```

Example

```
idem:refresh:123
```

## Purpose

클라이언트가 refresh 요청을 동시에 여러 번 보낼 경우
토큰 rotation이 꼬이는 문제를 방지합니다.

예:

- 네트워크 재전송
- 더블 클릭
- retry 로직

이러한 상황에서 **동시에 refresh가 실행되는 것을 방지**할 수 있습니다.

## Implementation

Redis `SET NX`를 사용합니다.

```
SET idem:refresh:{uid} 1 NX EX <short_ttl>
```

동작

| 상황 | 결과 |
|----|----|
| lock 성공 | refresh 진행 |
| lock 실패 | 이미 처리 중 → 요청 거부 |

TTL은 짧게 설정하여 이후 refresh는 막지 않도록 합니다.

예

```
5~10 seconds
```

---

# Login Rate Limit

```
rate:login:{ip}
```

Example

```
rate:login:192.168.1.10
```

## Purpose

로그인 brute force 공격을 방지하기 위해
IP 기반 rate limit을 적용합니다.

## Implementation

```
INCR rate:login:{ip}
EXPIRE rate:login:{ip} 60
```

예:

```
5 requests / minute
```

초과 시

```
HTTP 429
```

---

# Refresh Rate Limit

```
rate:refresh:{uid}
```

Example

```
rate:refresh:123
```

## Purpose

refresh endpoint abuse를 방지합니다.

Refresh 요청은 로그인 이후 반복적으로 호출될 수 있기 때문에
사용자 기준 rate limit을 적용합니다.

## Implementation

```
INCR rate:refresh:{uid}
EXPIRE rate:refresh:{uid} 60
```

예:

```
10 requests / minute
```

초과 시

```
HTTP 429
```

---

# Design Rationale

이 Redis 설계는 다음 보안 문제를 해결하기 위해 설계되었습니다.

| Problem | Solution |
|------|------|
| JWT logout 불가 | Redis session |
| refresh replay 공격 | refresh rotation |
| concurrent refresh | idempotency lock |
| login brute force | login rate limit |
| refresh abuse | refresh rate limit |

---

# Future Improvements

향후 다음과 같은 확장이 가능합니다.

- Redis replication / cluster 구성
- sliding session TTL
- refresh reuse detection 강화
- distributed rate limiting