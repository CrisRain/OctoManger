package accounttypedomain

import "time"

type AccountType struct {
	ID           int64          `json:"id"`
	Key          string         `json:"key"`
	Name         string         `json:"name"`
	Category     string         `json:"category"`
	Schema       map[string]any `json:"schema"`
	Capabilities map[string]any `json:"capabilities"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type CreateInput struct {
	Key          string         `json:"key"`
	Name         string         `json:"name"`
	Category     string         `json:"category"`
	Schema       map[string]any `json:"schema"`
	Capabilities map[string]any `json:"capabilities"`
}

type PatchInput struct {
	Name         *string        `json:"name,omitempty"`
	Category     *string        `json:"category,omitempty"`
	Schema       map[string]any `json:"schema,omitempty"`
	Capabilities map[string]any `json:"capabilities,omitempty"`
}
