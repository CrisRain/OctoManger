package accountdomain

import "time"

type Account struct {
	ID             int64          `json:"id"`
	AccountTypeID  *int64         `json:"account_type_id,omitempty"`
	AccountTypeKey string         `json:"account_type_key,omitempty"`
	Identifier     string         `json:"identifier"`
	Status         string         `json:"status"`
	Tags           []string       `json:"tags"`
	Spec           map[string]any `json:"spec"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type CreateInput struct {
	AccountTypeID int64          `json:"account_type_id"`
	Identifier    string         `json:"identifier"`
	Status        string         `json:"status"`
	Tags          []string       `json:"tags"`
	Spec          map[string]any `json:"spec"`
}

type PatchInput struct {
	Status *string        `json:"status,omitempty"`
	Tags   []string       `json:"tags,omitempty"`
	Spec   map[string]any `json:"spec,omitempty"`
}
