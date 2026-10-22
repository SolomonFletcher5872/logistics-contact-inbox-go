package logistics

import (
	"context"
	"fmt"
	"html"
	"strings"
)

type ShipmentEvent struct {
	At     string `json:"at"`
	Status string `json:"status"`
	Place  string `json:"place"`
}

type ProofOfDelivery struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

type Exception struct {
	Kind    string `json:"kind"`
	Details string `json:"details"`
}

type ContactForm struct {
	ShipmentID     string            `json:"shipment_id"`
	ReplyTo        string            `json:"reply_to"`
	Message        string            `json:"message"`
	Events         []ShipmentEvent   `json:"events"`
	Proofs         []ProofOfDelivery `json:"proofs"`
	Exceptions     []Exception       `json:"exceptions"`
	Captcha        string            `json:"captcha"`
	WidgetRecordID string            `json:"widget_record_id"`
}

type Gateway interface {
	VerifyCaptcha(context.Context, string, string, string) error
	SendEmail(context.Context, string, string, string, string) (EmailResult, error)
}

type Router struct {
	gateway Gateway
	inbox   string
}

func NewRouter(gateway Gateway, inbox string) *Router { return &Router{gateway: gateway, inbox: inbox} }

func (r *Router) Route(ctx context.Context, form ContactForm, clientIP string) (EmailResult, error) {
	if err := r.gateway.VerifyCaptcha(ctx, form.WidgetRecordID, form.Captcha, clientIP); err != nil {
		return EmailResult{}, err
	}
	return r.gateway.SendEmail(ctx, r.inbox, "Shipment "+form.ShipmentID+" needs review", renderDigest(form), form.ShipmentID)
}

func renderDigest(form ContactForm) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<h1>Shipment %s</h1><p>Reply to: %s</p><p>%s</p>", html.EscapeString(form.ShipmentID), html.EscapeString(form.ReplyTo), html.EscapeString(form.Message))
	b.WriteString("<h2>Events</h2><ul>")
	for _, event := range form.Events {
		fmt.Fprintf(&b, "<li>%s | %s | %s</li>", html.EscapeString(event.At), html.EscapeString(event.Status), html.EscapeString(event.Place))
	}
	b.WriteString("</ul><h2>Proof of delivery</h2><ul>")
	for _, proof := range form.Proofs {
		fmt.Fprintf(&b, "<li>%s: %s</li>", html.EscapeString(proof.Name), html.EscapeString(proof.Note))
	}
	b.WriteString("</ul><h2>Exceptions</h2><ul>")
	for _, exception := range form.Exceptions {
		fmt.Fprintf(&b, "<li>%s: %s</li>", html.EscapeString(exception.Kind), html.EscapeString(exception.Details))
	}
	b.WriteString("</ul>")
	return b.String()
}
