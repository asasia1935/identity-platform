# Error Handling Policy

## Overview

모든 API는 일관된 에러 응답을 제공해야 합니다.

---

## Status Codes

| 상황             | 상태 코드 |
| ---------------- | --------- |
| 인증 실패        | 401       |
| 권한 없음        | 403       |
| 잘못된 요청      | 400       |
| Rate Limit 초과  | 429       |
| Upstream Timeout | 504       |
| Upstream Error   | 502       |
| 서버 내부 오류   | 500       |

---

## Error Response Format

```json
{
  "error": "error_code"
}
```

## Principles

- 내부 구현 세부사항을 노출하지 않습니다.
- 단순하고 일관된 메시지를 유지합니다.
- 상태 코드로 의미를 전달합니다.

## Gateway Behavior

Gateway는 다음을 처리합니다:

- Upstream 오류 변환
- Timeout → 504
- 연결 실패 → 502

## Service Behavior

서비스는 다음을 처리합니다:

- 도메인 에러 처리
- 세션 및 Refresh 관련 에러 처리

## Examples

- invalid token → 401
- expired token → 401
- session 없음 → 401
- 잘못된 요청(JSON 오류 등) → 400
