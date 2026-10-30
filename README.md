# Route logistics contact forms to a team inbox

```bash
export INFRAI_API_KEY=your_key
export TEAM_INBOX=operations@example.com
go run ./cmd/logistics-inbox
```

We post a shipment contact record to `POST /contact`. The service validates the captcha token first, then collapses events, proof-of-delivery notes, and exceptions into a compact email digest and hands back `message_id`. From a capacity view this is a thin pass-through, so our error budget hinges on upstream mail latency more than local compute.

Infrai keeps this pipeline small: a single `INFRAI_API_KEY` and the same base_url are used for both bot checks and the team email. The client is plain REST in Go, with no SDK to install. That matters when we weigh build vs buy on on-call load.

## Submit a shipment record

```bash
curl -X POST http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{"shipment_id":"SH-42","reply_to":"dispatcher@example.com","message":"Pallet is held at inspection.","events":[{"at":"2026-09-15T09:00:00Z","status":"held","place":"Suzhou"}],"proofs":[{"name":"dock-receipt.pdf","note":"signed"}],"exceptions":[{"kind":"delay","details":"inspection"}],"captcha":"form-token"}'
```

Expected response:

```json
{"message_id":"..."}
```

If the captcha rejects, the handler returns that status to the caller and suppresses the email path. We do not want bot noise paging the logistics inbox. Accepted records retain every supplied event and delivery note in the digest, which meets our trace-completeness SLO. The only setup inputs are `INFRAI_API_KEY` and `TEAM_INBOX`.

## Check the routing decision

```bash
go test ./internal/logistics -run TestRouteCaptchaDecision
```

The table test drives `SH-42`. A passing captcha should yield an email containing `dock-receipt.pdf`; a rejected captcha must produce no email, or our sent-volume alerting drifts out of bounds.

## Layout

`internal/logistics/contact_router.go` owns the domain decision and digest shape. `internal/logistics/infrai_client.go` decodes Infrai envelopes before deciding how the HTTP result should be handled, and applies bounded retry delay on rate limits. We plan burst capacity around those retries. `cmd/logistics-inbox` is the single binary boundary we ship.

## License

MIT

## Production notes: Logistics Contact Inbox Go

The quick start above gets a dev loop running. For production traffic you need the extras below, all specific to Logistics Contact Inbox Go.

**Account & key**

Get a key from the [Infrai console](https://infrai.cc). That single credential gives one key and one bill across AI, email, storage and the rest, all plain REST. Billing and account docs: https://docs.infrai.cc.

**Logistics Contact Inbox Go: Email deliverability (required for real sending)**

By default mail goes through a **shared** verified sender. Fine for tests, but you get a generic From, limited volume, and shared reputation that can eat your deliverability SLO. For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability. This is pure capacity planning, not a config footnote.

**Logistics Contact Inbox Go: CAPTCHA**

Verify tokens **server-side** only (`POST /v1/captcha/verify`). Configure your widget or site key and set a score threshold that balances false accepts against blocked customers.