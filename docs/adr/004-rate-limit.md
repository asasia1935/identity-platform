# ADR-004: Rate Limiting for Authentication Endpoints

Status: Accepted

## Context

인증 엔드포인트는 공격에 매우 취약합니다.

대표적인 공격:

- credential stuffing
- brute force login
- refresh token abuse

따라서 인증 API에는 요청 빈도 제한이 필요할 수밖에 없습니다.

## Decision

Redis 기반 Rate Limiting을 적용합니다.

대상 엔드포인트:

- `/auth/login`
- `/auth/refresh`

Redis key:

```
rate:login:{ip}
rate:refresh:{uid}
```

TTL 기반 카운터 방식으로 구현한다.

예:

- login: 5 requests / 10 seconds
- refresh: 10 requests / 10 seconds

Rate limit 초과 시 HTTP 429 반환합니다.

## Consequences

### Positive

- brute force 공격 완화
- refresh abuse 방지
- 인증 인프라 보호

### Negative

- Redis 의존성 증가
- 일부 정상 사용자 요청이 제한될 가능성