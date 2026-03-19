# ADR-003: Refresh Idempotency

Status: Accepted

## Context

클라이언트 환경에서는 다음 상황이 발생할 수 있습니다.

- 네트워크 재시도
- 모바일 앱 중복 요청
- 동일 refresh 요청의 동시 실행

Refresh Rotation이 적용된 환경에서는  
동일 요청이 동시에 처리되면 다음 문제가 발생합니다.

- 첫 요청이 rotation 수행
- 두 번째 요청이 이전 토큰을 사용하여 실패
- 정상 사용자에게 인증 오류 발생

이를 방지하기 위해 Refresh 요청은 idempotent하게 처리되어야 합니다.

## Decision

Redis 기반 idempotency lock을 도입합니다.

Refresh 요청 시:

```
SETNX idem:refresh:{uid}
```

락 획득에 실패하면 동일 요청으로 판단하고 거부합니다.

락 TTL은 짧게 설정합니다.

예:

```
refresh_idempotency_ttl = 3s
```

## Consequences

### Positive

- refresh race condition 방지
- 중복 refresh 요청 안전 처리
- 모바일 네트워크 재시도 안정성 확보

### Negative

- Redis lock 관리 필요
- refresh 요청 경로에 추가 Redis 호출 발생