# app-feedback

Temporary survey API for **Neuland Next** feedback campaigns.

Deliberately **not** part of Backend 2.0 — spin it up for a campaign, export the
results, tear it down. CI/CD mirrors [`discord-bot`](https://github.com/neuland-ingolstadt/discord-bot).

## Endpoints

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/healthz` | — | Liveness + SQLite ping |
| `PUT` | `/v1/feedback` | optional `X-Feedback-Key` | Upsert a feedback submission |
| `GET` | `/v1/admin/export` | `Authorization: Bearer $ADMIN_TOKEN` | Dump stored rows |

### Submit body

Matches the app payload from PR #810:

```json
{
  "id": "client-generated-id",
  "campaign": "feedback-campaign-2026-10-29_18-55",
  "rating": 4,
  "categories": ["timetable"],
  "comment": "optional",
  "featureInterest": { "digitalId": 3 },
  "missingFeatures": "optional",
  "platform": "ios",
  "appVersion": "1.2.3"
}
```

## Guardrails (kept simple)

- Per-IP and per-submission-id rate limits
- Upsert budget per id (`MAX_UPSERTS_PER_ID`)
- Payload validation + optional campaign allowlist
- Optional shared write key (`WRITE_KEY`)

## Local run

```bash
cp .env.example .env
make build
./bin/server
```

```bash
curl -s -X PUT localhost:8080/v1/feedback \
  -H 'content-type: application/json' \
  -d '{"id":"abc12345","campaign":"feedback-campaign-2026-10-29_18-55","rating":5,"platform":"ios"}'
```

## Docker / CI

- `ci.yml` — tidy, gofmt, vet, race tests, build
- `build-docker.yml` — image to `ghcr.io/neuland-ingolstadt/app-feedback`

```bash
make docker-build
```

## Wire into the app later

Replace the stub in `submitFeedback` with `PUT {FEEDBACK_API}/v1/feedback`
using the existing payload shape.
