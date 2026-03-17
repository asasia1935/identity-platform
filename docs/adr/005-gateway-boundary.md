# ADR-005: Gateway Boundary Enforcement

Status: Accepted

## Context

Identity 플랫폼은 내부 서비스로 직접 접근하는 것을 허용하면 안됩니다.

문제점:

- 인증 정책 우회 가능
- 내부 API 직접 호출 가능
- 보안 경계 붕괴

따라서 모든 외부 요청은 반드시 Gateway를 통해서만 진입해야 합니다.

## Decision

Gateway Boundary Header를 도입합니다.

Gateway는 내부 서비스로 요청을 전달할 때 다음 헤더를 추가합니다.

```
X-Gateway-Verified: true
```

Auth 서비스는 이 헤더가 없는 요청을 거부합니다.

즉:

```
if header missing → reject request
```

이를 통해 내부 서비스 직접 접근을 차단합니다.

## Consequences

### Positive

- 인증 경계 명확화
- 내부 서비스 직접 호출 차단
- Gateway 정책 강제

### Negative

- 서비스 간 호출 시 게이트웨이 헤더 의존성 발생