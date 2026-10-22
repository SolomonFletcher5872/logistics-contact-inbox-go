package logistics

import (
	"context"
	"strings"
	"testing"
)

type fakeGateway struct {
	verifyErr      error
	emailSent      bool
	html           string
	widgetRecordID string
}

func (f *fakeGateway) VerifyCaptcha(_ context.Context, widgetRecordID, _, _ string) error {
	f.widgetRecordID = widgetRecordID
	return f.verifyErr
}

func (f *fakeGateway) SendEmail(_ context.Context, _ string, _ string, html, _ string) (EmailResult, error) {
	f.emailSent, f.html = true, html
	return EmailResult{MessageID: "msg_123"}, nil
}

func TestRouteCaptchaDecision(t *testing.T) {
	form := ContactForm{ShipmentID: "SH-42", ReplyTo: "ops@example.test", Message: "Pallet held", Events: []ShipmentEvent{{At: "2026-09-15T09:00:00Z", Status: "held", Place: "Suzhou"}}, Proofs: []ProofOfDelivery{{Name: "dock-receipt.pdf", Note: "signed"}}, Exceptions: []Exception{{Kind: "delay", Details: "inspection"}}, Captcha: "token", WidgetRecordID: "widget_123"}
	for _, tc := range []struct {
		name, contains string
		wantErr        bool
	}{
		{name: "accepted form becomes inbox digest", contains: "dock-receipt.pdf"},
		{name: "rejected captcha does not email", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeGateway{}
			if tc.wantErr {
				fake.verifyErr = &InfraiError{Message: "captcha did not pass", StatusCode: 422}
			}
			_, err := NewRouter(fake, "team@example.test").Route(context.Background(), form, "203.0.113.8")
			if (err != nil) != tc.wantErr || fake.emailSent == tc.wantErr || fake.widgetRecordID != form.WidgetRecordID || (!tc.wantErr && !strings.Contains(fake.html, tc.contains)) {
				t.Fatalf("err=%v emailSent=%v html=%q", err, fake.emailSent, fake.html)
			}
		})
	}
}
