# ADR-001: Redis-based Session Management

Status: Accepted

## Context

JWT 기반 인증은 Stateless 방식으로,
서버에서 별도의 상태 저장 없이 토큰만으로 인증을 처리할 수 있습니다.

하지만 순수 Stateless JWT만 사용할 경우 다음 문제가 발생합니다.

- 서버에서 로그인 상태를 즉시 무효화할 수 없음 (logout 불가)
- 토큰 탈취 시 서버에서 차단할 수 없음
- 인증 상태를 중앙에서 통제할 수 없음

본 Identity 플랫폼은 MSA 환경에서 공용 인증 인프라로 동작하며,
Gateway가 Access Token 검증을 담당하는 구조를 가집니다.

이 구조에서는 내부 서비스가 JWT를 직접 검증하지 않기 때문에,
“현재 로그인 상태인지”를 판단할 수 있는 별도의 서버 측 상태 저장소가 필요합니다.

## Decision

세션 저장소로 Redis를 사용합니다.

- key: `sess:{uid}`
- value: session marker (e.g. `"1"`)
- TTL: Access Token TTL과 동일하게 설정

Redis에는 세션의 상세 상태를 저장하지 않고,
key의 존재 여부만 로그인 상태의 기준으로 사용합니다.

### 인증 흐름

Protected API 접근 시:

1. Gateway가 Access Token을 검증합니다.
2. Gateway는 검증된 사용자 식별 정보를 내부 요청에 전달합니다.
3. 내부 서비스(Auth)는 전달받은 사용자 식별자를 기준으로 Redis session 존재 여부를 확인합니다.

이때 Redis에 세션이 존재하지 않으면 인증 실패로 처리합니다.

## Consequences

### Positive

- 서버에서 즉시 세션 무효화 가능 (logout)
- 토큰 탈취 시 세션 삭제로 즉시 차단 가능
- Gateway와 역할 분리가 명확해짐 (검증 vs 상태 확인)
- Redis TTL 기반 자동 세션 정리

### Negative

- 보호 API의 인증 성공 여부가 Redis session 조회에 의존
- Redis 장애 시 인증 요청이 지연되거나 실패할 수 있음

Gateway timeout 정책은 장애 상황에서 지연 요청을 빠르게 차단하여 응답 지연 확산을 완화할 수 있습니다.