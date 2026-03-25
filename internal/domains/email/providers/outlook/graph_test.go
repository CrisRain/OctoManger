package outlook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGraphMailboxAndMessageFlow(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("unexpected authorization header %q", got)
		}

		switch r.URL.Path {
		case "/me/mailFolders":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"value": []map[string]any{
					{"id": "inbox", "displayName": "Inbox"},
					{"id": "alerts", "displayName": "Alerts"},
				},
				"@odata.count": 2,
			})
		case "/me/mailFolders/Alerts", "/me/mailFolders/alerts":
			http.Error(w, `{"error":{"code":"not_found","message":"missing"}}`, http.StatusNotFound)
		case "/me/mailFolders/alerts/messages":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"value": []map[string]any{
					{
						"id":               "msg-1",
						"subject":          "Deploy finished",
						"from":             map[string]any{"emailAddress": map[string]any{"address": "build@example.com", "name": "Build Bot"}},
						"toRecipients":     []map[string]any{{"emailAddress": map[string]any{"address": "robot@example.com", "name": "Robot"}}},
						"receivedDateTime": "2026-03-14T10:00:00Z",
						"size":             128,
						"isRead":           true,
					},
				},
				"@odata.count": 1,
			})
		case "/me/messages/msg-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":               "msg-1",
				"subject":          "Deploy finished",
				"from":             map[string]any{"emailAddress": map[string]any{"address": "build@example.com", "name": "Build Bot"}},
				"toRecipients":     []map[string]any{{"emailAddress": map[string]any{"address": "robot@example.com", "name": "Robot"}}},
				"ccRecipients":     []map[string]any{{"emailAddress": map[string]any{"address": "ops@example.com", "name": "Ops"}}},
				"receivedDateTime": "2026-03-14T10:00:00Z",
				"size":             128,
				"isRead":           true,
				"internetMessageHeaders": []map[string]any{
					{"name": "Message-ID", "value": "<msg-1@example.com>"},
				},
				"body": map[string]any{
					"contentType": "text",
					"content":     "Deployment finished successfully.",
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	config := GraphConfig{
		AccessToken: "token-123",
		BaseURL:     server.URL,
	}

	mailboxes, err := ListMailboxes(context.Background(), config, 200)
	if err != nil {
		t.Fatalf("list mailboxes: %v", err)
	}
	if len(mailboxes) != 2 {
		t.Fatalf("expected 2 mailboxes, got %d", len(mailboxes))
	}

	mailboxID, mailboxName, err := ResolveMailbox(context.Background(), config, "Alerts")
	if err != nil {
		t.Fatalf("resolve mailbox: %v", err)
	}
	if mailboxID != "alerts" || mailboxName != "Alerts" {
		t.Fatalf("unexpected mailbox resolution %q / %q", mailboxID, mailboxName)
	}

	messages, total, err := ListMessages(context.Background(), config, mailboxID, 20, 0)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if total != 1 || len(messages) != 1 {
		t.Fatalf("unexpected list message result total=%d items=%d", total, len(messages))
	}
	if messages[0].Date != time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC) {
		t.Fatalf("unexpected message date %s", messages[0].Date)
	}

	message, err := GetMessage(context.Background(), config, "msg-1")
	if err != nil {
		t.Fatalf("get message: %v", err)
	}
	if message.Subject != "Deploy finished" {
		t.Fatalf("unexpected subject %q", message.Subject)
	}
	if message.TextBody != "Deployment finished successfully." {
		t.Fatalf("unexpected text body %q", message.TextBody)
	}
	if message.Headers["Message-ID"] != "<msg-1@example.com>" {
		t.Fatalf("unexpected Message-ID header %q", message.Headers["Message-ID"])
	}
}
