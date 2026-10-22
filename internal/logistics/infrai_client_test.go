package logistics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyCaptchaRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/captcha/verify" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["widget_record_id"] != "widget_123" || payload["token"] != "token_123" {
			t.Errorf("unexpected captcha payload: %v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{}}`))
	}))
	defer server.Close()
	client := &Client{baseURL: server.URL, key: "test", http: server.Client()}
	if err := client.VerifyCaptcha(context.Background(), "widget_123", "token_123", "203.0.113.8"); err != nil {
		t.Fatal(err)
	}
}
