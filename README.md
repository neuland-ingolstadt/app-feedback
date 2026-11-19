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
| `GET` | `/metrics` | — | Prometheus gauges derived from SQLite |

`/healthz` and `/v1/feedback` are the only routes on the public ingress
(`survey.neuland.app`). `/v1/admin/export` and `/metrics` stay on the process
so they can be reached inside the cluster, and are not routed publicly.

### Admin export

The handler is unchanged. Fetch the raw rows with a port-forward (the bearer
token is the `ADMIN_TOKEN` key in the `app-feedback-secrets` secret). Do not
put that token in Grafana.

```bash
kubectl -n neuland-app port-forward svc/app-feedback 8080:8080
curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" http://127.0.0.1:8080/v1/admin/export
```

### Metrics

`GET /metrics` recomputes gauges from the current SQLite rows on each scrape:
submission totals, rating sum and histogram, platform, category, feature
interest (count per level and score sum), and counts of non-empty comments and
missing-feature notes. Free text is not exported. VictoriaMetrics scrapes the
ClusterIP service; nothing on the public ingress should forward `/metrics`.

### Submit body

Matches the app payload from neuland.app-native PR #839:

```json
{
  "id": "client-generated-id",
  "campaign": "feedback-campaign-2026-10-29_18-55",
  "rating": 4,
  "categories": ["timetable"],
  "comment": "optional",
  "featureInterest": {
    "digitalId": 3,
    "mensaPayment": 2,
    "librarySeats": 4,
    "roomNavigation": 1,
    "studentExchange": 3,
    "campusUpdates": 4,
    "eventSignup": 2,
    "mensaRatings": 3
  },
  "platform": "ios",
  "appVersion": "1.2.3"
}
```

`missingFeatures` is still accepted for older clients but the current survey UI
only sends a single free-text `comment`.

## Guardrails (kept simple)

- Per-IP and per-submission-id rate limits (IP only in memory, **never persisted**)
- Upsert budget per id (`MAX_UPSERTS_PER_ID`)
- Payload validation + optional campaign allowlist
- Optional shared write key (`WRITE_KEY`)

Stored rows are survey answers only: rating, categories, optional free text,
feature interest, platform, app version, timestamps. No IP, no account link.

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
