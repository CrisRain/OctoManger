package emaildomain

import "time"

const (
	StatusPending  = "pending"
	StatusActive   = "active"
	StatusInactive = "inactive"
)

type Account struct {
	ID        int64          `json:"id"`
	Address   string         `json:"address"`
	Provider  string         `json:"provider"`
	Status    string         `json:"status"`
	Config    map[string]any `json:"config"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CreateInput struct {
	Address  string         `json:"address"`
	Provider string         `json:"provider"`
	Status   string         `json:"status"`
	Config   map[string]any `json:"config"`
}

type PatchInput struct {
	Provider *string        `json:"provider,omitempty"`
	Status   *string        `json:"status,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

type ListMailboxesInput struct {
	Pattern string `json:"pattern,omitempty"`
}

type Mailbox struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListMailboxesResult struct {
	Pattern string    `json:"pattern,omitempty"`
	Items   []Mailbox `json:"items"`
}

type ListMessagesInput struct {
	Mailbox string `json:"mailbox,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

type MessageSummary struct {
	ID      string    `json:"id"`
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	Date    time.Time `json:"date"`
	Size    int64     `json:"size"`
	Flags   []string  `json:"flags"`
}

type ListMessagesResult struct {
	Mailbox string           `json:"mailbox"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
	Total   int              `json:"total"`
	Items   []MessageSummary `json:"items"`
}

type MessageDetail struct {
	ID       string            `json:"id"`
	Subject  string            `json:"subject"`
	From     string            `json:"from"`
	To       string            `json:"to"`
	Cc       string            `json:"cc"`
	Date     time.Time         `json:"date"`
	Size     int64             `json:"size"`
	Flags    []string          `json:"flags"`
	Headers  map[string]string `json:"headers"`
	TextBody string            `json:"text_body"`
	HTMLBody string            `json:"html_body"`
}

type LatestMessageResult struct {
	Mailbox string         `json:"mailbox"`
	Found   bool           `json:"found"`
	Item    *MessageDetail `json:"item,omitempty"`
}

type OutlookAuthorizeURLResult struct {
	AuthorizeURL string `json:"authorize_url"`
}

type OutlookExchangeCodeInput struct {
	Code string `json:"code"`
}

type PreviewInput struct {
	Config  map[string]any `json:"config"`
	Mailbox string         `json:"mailbox,omitempty"`
	Pattern string         `json:"pattern,omitempty"`
}

type BulkImportInput struct {
	Lines []string `json:"lines"`
}

type BulkImportLineResult struct {
	Line    string `json:"line"`
	Address string `json:"address,omitempty"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	ID      *int64 `json:"id,omitempty"`
}

type BulkImportResult struct {
	Total   int                    `json:"total"`
	Success int                    `json:"success"`
	Failed  int                    `json:"failed"`
	Items   []BulkImportLineResult `json:"items"`
}
