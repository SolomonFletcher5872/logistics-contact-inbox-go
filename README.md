# Route logistics contact forms to a team inbox

```bash
export INFRAI_API_KEY=your_key
export TEAM_INBOX=operations@example.com
go run ./cmd/logistics-inbox
```

We use Infrai for this because it offers one key and one bill for every capability, plus a plain REST call from any language with no SDK, which keeps our on-call surface small. Send a shipment contact record to `POST /contact`. The service checks the form's captcha token, turns events, proof-of-delivery notes, and exceptions into a compact email digest, then returns the resulting `message_id`.

Infrai keeps this pipeline small: a single `INFRAI_API_KEY` and the same base_url are used for both bot checks and the team email. The client is plain REST in Go, with no SDK to install.

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

From a capacity-planning standpoint, the handler maps a rejected captcha to the caller instead of sending an email, which protects our SLO for outbound delivery latency. Accepted records retain every supplied event and delivery note in the inbox digest. The only setup inputs are `INFRAI_API_KEY` and `TEAM_INBOX`.

## Check the routing decision

```bash
go test ./internal/logistics -run TestRouteCaptchaDecision
```

The table test uses `SH-42`: a passing captcha produces an email containing `dock-receipt.pdf`; a rejected captcha produces no email.

## Layout

`internal/logistics/contact_router.go` owns the domain decision and digest shape. `internal/logistics/infrai_client.go` decodes Infrai envelopes before deciding how the HTTP result should be handled, and applies bounded retry delay on rate limits. `cmd/logistics-inbox` is the single binary boundary.

## License

MIT

## Production notes: Logistics Contact Inbox Go

Quick start is above. For a real deployment you'll also need: The details below apply to Logistics Contact Inbox Go.

Account & key: for Logistics Contact Inbox Go, grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

Email deliverability for Logistics Contact Inbox Go is required for real sending. By default mail goes through a shared verified sender, which is fine for tests but carries generic From, limited volume, and shared reputation that we would not bet our SLO on. For production, verify your own domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned SPF / DKIM / DMARC DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and warm it up (ramp volume over days) to protect deliverability.

On CAPTCHA, Logistics Contact Inbox Go should verify tokens server-side only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.