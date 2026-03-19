# Auth Boundary Policy

## Overview

본 시스템은 인증(Authentication)과 권한(Authorization)의 책임을 분리하여 설계합니다.

- 인증: Gateway에서 수행합니다.
- 권한 및 비즈니스 로직: 각 서비스에서 수행합니다.

---

## Gateway Responsibilities

Gateway는 인증의 단일 진입점입니다.

다음 역할을 수행합니다:

- JWT Access Token 검증
- 토큰 만료 및 서명 검증
- 인증 실패 시 401 응답 반환
- 사용자 식별자 추출
- `X-User-ID` 헤더 주입
- `X-Gateway-Verified` 헤더 주입

Gateway를 통과하지 않은 요청은 신뢰하지 않습니다.

---

## Service Responsibilities

각 서비스(Auth 포함)는 다음을 수행합니다:

- Gateway에서 전달한 사용자 컨텍스트를 사용합니다.
- JWT를 직접 검증하지 않습니다.
- 도메인 수준의 권한 검사를 수행합니다.

---

## Trust Model

서비스는 다음 조건을 만족할 때만 요청을 신뢰합니다:

- `X-Gateway-Verified: true` 헤더 존재
- `X-User-ID` 헤더 존재

위 조건이 충족되지 않을 경우 요청을 거부해야 합니다.

---

## Rationale

본 구조는 다음과 같은 문제를 해결합니다:

- 서비스별 JWT 검증 중복 제거
- 인증 정책의 중앙화
- 유지보수 비용 감소

---

## Trade-off

- Gateway 장애 발생 시 인증이 불가능합니다.
- 내부 네트워크에 대한 신뢰가 필요합니다.

향후 mTLS 또는 service mesh 도입으로 보완할 수 있습니다.
