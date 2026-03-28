package emailapp

import (
	"testing"

	"octomanger/internal/domains/email/providers/outlook"
)

func TestStatusFromEmailAuthFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config outlook.AccountConfig
		want   string
	}{
		{
			name:   "missing credentials stays pending",
			config: outlook.AccountConfig{},
			want:   "pending",
		},
		{
			name: "existing token failure becomes inactive",
			config: outlook.AccountConfig{
				RefreshToken: "refresh-token",
			},
			want: "inactive",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := statusFromEmailAuthFailure(tt.config); got != tt.want {
				t.Fatalf("status = %q, want %q", got, tt.want)
			}
		})
	}
}
