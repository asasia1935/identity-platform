# ADR-006: Short-lived Access Token Policy

Status: Accepted

## Context

JWT Access Token은 Stateless 인증 방식으로 서버에서 별도의 저장 없이 검증할 수 있습니다.

하지만 Access Token이 탈취될 경우 다음 문제가 발생합니다.

- 공격자가 해당 토큰을 이용해 Protected API에 접근 가능
- 서버 측에서 토큰을 즉시 무효화하기 어려움
- 토큰이 유효한 동안 공격이 지속될 수 있음

Stateless JWT 구조에서는 이미 발급된 Access Token을 서버에서 직접 폐기할 수 없기 때문에  
토큰 탈취 리스크를 줄이기 위한 전략이 필요합니다.

## Decision

Access Token을 **Short-lived Token**으로 운영합니다.

예:

```
access_token_ttl = 15m
```

Access Token은 짧은 수명을 가지며  
Access Token이 만료되면 Refresh Token을 사용하여 재발급하도록 합니다.

인증 흐름은 다음과 같다.

1. Client login
2. Access Token + Refresh Token 발급
3. Access Token으로 API 호출
4. Access Token 만료 시 Refresh Token으로 재발급

이 구조에서 Access Token은 **빈번하게 교체되는 토큰**이며  
장기 인증 상태는 Refresh Token이 담당합니다.

## Consequences

### Positive

- Access Token 탈취 시 공격 지속 시간을 제한할 수 있음
- 토큰 수명이 짧아 보안성이 향상됨
- 서버에서 별도 저장 없이 빠른 JWT 검증 가능

### Negative

- Access Token 만료로 인해 refresh 요청이 증가할 수 있음
- refresh endpoint 부하 가능성 존재

이를 완화하기 위해 다음 정책을 함께 적용합니다.

- Refresh Token Rotation
- Refresh Idempotency
- Rate Limiting
- Redis Session 관리

이 정책들을 통해 Refresh 경로의 안정성과 보안을 동시에 확보합니다.