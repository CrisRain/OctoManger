package runtime

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"octomanger/internal/platform/database"
	"octomanger/internal/testutil"
)

func TestTrimGroupedJobLogsNoRows(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	deleted, err := trimGroupedJobLogs(ctx, db, 1)
	if err != nil {
		t.Fatalf("trim grouped job logs: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected no deletions, got %d", deleted)
	}
}

func TestTrimGroupedJobLogsDeletes(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	seedJobLogs(t, db, 1, startupJobLogRetentionLimit+1)

	deleted, err := trimGroupedJobLogs(ctx, db, 1)
	if err != nil {
		t.Fatalf("trim grouped job logs: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deletion, got %d", deleted)
	}
}

func TestTrimGroupedAgentLogsDeletes(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	seedAgentLogs(t, db, 1, startupAgentLogRetentionLimit+1)

	deleted, err := trimGroupedAgentLogs(ctx, db, 1)
	if err != nil {
		t.Fatalf("trim grouped agent logs: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deletion, got %d", deleted)
	}
}

func TestEnforceLogRetentionOnStartupHandlesErrors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	app := &App{Logger: zap.NewNop(), DB: db}
	enforceLogRetentionOnStartup(context.Background(), app)
}

func seedJobLogs(t *testing.T, db *gorm.DB, executionID int64, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		record := database.JobLogModel{
			JobExecutionID: executionID,
			Stream:         "plugin",
			EventType:      "log",
			Message:        "test",
			PayloadJSON:    database.JSONBytes([]byte("{}")),
			CreatedAt:      time.Now().UTC(),
		}
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("insert job log: %v", err)
		}
	}
}

func seedAgentLogs(t *testing.T, db *gorm.DB, agentID int64, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		record := database.AgentLogModel{
			AgentID:     agentID,
			EventType:   "log",
			Message:     "test",
			PayloadJSON: database.JSONBytes([]byte("{}")),
			CreatedAt:   time.Now().UTC(),
		}
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("insert agent log: %v", err)
		}
	}
}
