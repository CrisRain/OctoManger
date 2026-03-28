package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// EnsureSystemLogSchema creates the runtime log table/indexes when they are
// missing so the app can start emitting logs even before a full migration run.
func EnsureSystemLogSchema(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.WithContext(ctx).AutoMigrate(&SystemLogModel{}); err != nil {
		return fmt.Errorf("ensure system log schema: %w", err)
	}
	return nil
}
