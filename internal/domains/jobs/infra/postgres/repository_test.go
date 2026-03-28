package jobpostgres

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"

	jobdomain "octomanger/internal/domains/jobs/domain"
	platformconfig "octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
)

func TestRepositoryListExecutionsPageAndDeleteDefinitionCascade(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("OCTOMANGER_TEST_DATABASE_DSN"))
	if dsn == "" {
		t.Skip("OCTOMANGER_TEST_DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db := openSchemaDB(t, ctx, dsn)
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	repo := New(db)

	def, err := repo.CreateDefinition(ctx, jobdomain.CreateDefinitionInput{
		Key:       "repo-test-definition",
		Name:      "Repo Test Definition",
		PluginKey: "octo_demo",
		Action:    "LIST_TASKS",
		Input:     map[string]any{"seed": true},
		Schedule: &jobdomain.ScheduleInput{
			CronExpression: "*/5 * * * *",
			Timezone:       "UTC",
			Enabled:        true,
		},
	}, nil)
	if err != nil {
		t.Fatalf("create definition: %v", err)
	}

	for i := 0; i < 3; i++ {
		exec, err := repo.EnqueueExecution(ctx, def.ID, "test", "manual", map[string]any{"n": i})
		if err != nil {
			t.Fatalf("enqueue execution %d: %v", i, err)
		}
		if err := repo.AppendLog(ctx, exec.ID, "plugin", "result", fmt.Sprintf("log-%d", i), map[string]any{"n": i}); err != nil {
			t.Fatalf("append log %d: %v", i, err)
		}
	}

	if err := db.WithContext(ctx).Create(&database.TriggerModel{
		Key:              "repo-test-trigger",
		Name:             "Repo Test Trigger",
		JobDefinitionID:  def.ID,
		Mode:             "async",
		DefaultInputJSON: database.JSONBytes([]byte("{}")),
		TokenHash:        "hash",
		TokenPrefix:      "hash",
		Enabled:          true,
	}).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	pageItems, total, err := repo.ListExecutionsPage(ctx, 2, 0)
	if err != nil {
		t.Fatalf("list executions page: %v", err)
	}
	if total != 3 {
		t.Fatalf("unexpected total executions %d", total)
	}
	if len(pageItems) != 2 {
		t.Fatalf("unexpected page item count %d", len(pageItems))
	}

	if err := repo.DeleteDefinition(ctx, def.ID); err != nil {
		t.Fatalf("delete definition with cascade cleanup: %v", err)
	}

	assertCount := func(table string, want int64) {
		var count int64
		if err := db.WithContext(ctx).Table(table).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != want {
			t.Fatalf("unexpected %s count %d, want %d", table, count, want)
		}
	}

	assertCount("job_definitions", 0)
	assertCount("schedules", 0)
	assertCount("job_executions", 0)
	assertCount("job_logs", 0)
	assertCount("triggers", 0)
}

func openSchemaDB(t *testing.T, ctx context.Context, baseDSN string) *gorm.DB {
	t.Helper()

	adminPool, err := pgxpool.New(ctx, baseDSN)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)

	schemaName := fmt.Sprintf("octomanger_jobs_repo_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName))
	})

	db, err := database.Open(platformconfig.DatabaseConfig{
		DSN:              withSearchPath(baseDSN, schemaName),
		MigrationMode:    "versioned",
		MaxConnections:   4,
		QueryTimeout:     5 * time.Second,
		HealthcheckGrace: 250 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("open schema db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

func withSearchPath(baseDSN, schemaName string) string {
	if strings.Contains(baseDSN, "://") {
		parsed, err := url.Parse(baseDSN)
		if err == nil {
			query := parsed.Query()
			query.Set("search_path", schemaName)
			parsed.RawQuery = query.Encode()
			return parsed.String()
		}
	}
	return strings.TrimSpace(baseDSN) + " search_path=" + schemaName
}
