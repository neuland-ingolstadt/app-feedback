# app-feedback

Temporary survey API for **Neuland Next** feedback campaigns.

Deliberately **not** part of Backend 2.0 — spin it up for a campaign, export the
results, tear it down. CI/CD mirrors [`discord-bot`](https://github.com/neuland-ingolstadt/discord-bot).

## Endpoints

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/healthz` | — | Liveness + SQLite ping |
| `GET` | `/v1/challenge` | optional `X-Feedback-Key` | Issue a signed PoW challenge |
| `PUT` | `/v1/feedback` | optional `X-Feedback-Key` | Upsert a feedback submission |
| `GET` | `/v1/admin/export` | `Authorization: Bearer $ADMIN_TOKEN` | Dump stored rows (`?shadow=1` includes shadowbans) |

### Submit body

Matches the app payload from PR #810, plus abuse fields:

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
  "appVersion": "1.2.3",
  "challenge": "<opaque token from /v1/challenge>",
  "counter": 123456,
  "website": ""
}
```

`website` is a honeypot — real clients leave it empty. Bots that fill it get
`204` but the row is shadow-banned.

## Manipulation controls (yes, overengineered)

1. **Proof of work** — `GET /v1/challenge` returns a nonce; client must find
   `counter` such that `SHA256(nonce:counter)` has `POW_DIFFICULTY` leading zero
   bits. Token is HMAC-signed and IP-class bound.
2. **Replay cache** — each challenge id can be spent once (memory + SQLite).
3. **Rate limits** — token buckets per IP and per submission id.
4. **Upsert budget** — max N mutations per id (`MAX_UPSERTS_PER_ID`).
5. **Honeypot** — `website` field → silent shadowban.
6. **Trust score** — velocity, duplicate comments, spam heuristics, PoW timing,
   rating flips. High score → shadowban or hard reject.
7. **Optional write key** — `WRITE_KEY` / `X-Feedback-Key`.
8. **Campaign allowlist** — `ALLOWED_CAMPAIGNS`.

Shadowbans still return `204` so scrapers learn nothing.

## Local run

```bash
cp .env.example .env
make build
ALLOW_INSECURE_DEV_SECRET=1 ./bin/server
```

```bash
# 1) challenge
curl -s localhost:8080/v1/challenge | jq

# 2) mine counter (tests use abuse.Solve; ship a tiny helper in the app)
# 3) submit
curl -s -X PUT localhost:8080/v1/feedback \
  -H 'content-type: application/json' \
  -d '{"id":"abc12345","campaign":"feedback-campaign-2026-10-29_18-55","rating":5,"challenge":"...","counter":0}'
```

## Docker / CI

- `ci.yml` — tidy, gofmt, vet, race tests, build (same shape as discord-bot)
- `build-docker.yml` — multi-arch image to `ghcr.io/neuland-ingolstadt/app-feedback`

```bash
make docker-build
```

## Wire into the app later

Replace the stub in `submitFeedback` with:

1. `GET {FEEDBACK_API}/v1/challenge`
2. mine PoW in JS (Web Worker / native)
3. `PUT {FEEDBACK_API}/v1/feedback` with the existing payload + `challenge`/`counter`
