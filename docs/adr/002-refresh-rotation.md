# ADR-002: Refresh Token Rotation

Status: Accepted

## Context

Refresh Token을 고정적으로 사용하는 경우 다음 문제가 발생합니다.

- 탈취된 Refresh Token이 장기간 재사용될 수 있음
- 공격자가 Access Token을 계속 발급받을 수 있음

Identity 플랫폼은 토큰 탈취 가능성을 전제로 설계되어야 합니다.

## Decision

Refresh Token Rotation 전략을 적용합니다.

Refresh 요청 시:

1. 기존 refresh token 검증
2. 기존 refresh jti 삭제
3. 새로운 refresh token 발급
4. 새로운 refresh jti 저장

Redis key 구조:

```
rjti:{uid}
```

Refresh Token은 매 요청마다 새로운 토큰으로 교체됩니다.

## Consequences

### Positive

- Refresh token 재사용 공격 방지
- 탈취 토큰의 사용 기간을 최소화
- 토큰 수명 단축 효과

### Negative

- Redis 상태 관리 필요
- 동시 refresh 요청 시 race condition 발생 가능

이를 해결하기 위해 Refresh Idempotency 정책을 추가해야 합니다.