package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	accounttypestransport "octomanger/internal/domains/account-types/transport"
	accountapp "octomanger/internal/domains/accounts/app"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	accountstransport "octomanger/internal/domains/accounts/transport"
	agentapp "octomanger/internal/domains/agents/app"
	agentpostgres "octomanger/internal/domains/agents/infra/postgres"
	agenttransport "octomanger/internal/domains/agents/transport"
	emailapp "octomanger/internal/domains/email/app"
	emailpostgres "octomanger/internal/domains/email/infra/postgres"
	emailtransport "octomanger/internal/domains/email/transport"
	jobapp "octomanger/internal/domains/jobs/app"
	jobpostgres "octomanger/internal/domains/jobs/infra/postgres"
	jobtransport "octomanger/internal/domains/jobs/transport"
	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/domains/plugins/infra/fsrepo"
	plugintransport "octomanger/internal/domains/plugins/transport"
	systemapp "octomanger/internal/domains/system/app"
	systemtransport "octomanger/internal/domains/system/transport"
	triggerapp "octomanger/internal/domains/triggers/app"
	triggerpostgres "octomanger/internal/domains/triggers/infra/postgres"
	triggertransport "octomanger/internal/domains/triggers/transport"
	platformconfig "octomanger/internal/platform/config"
	"octomanger/internal/platform/logging"
	"octomanger/internal/platform/migrations"
)

func TestAPISmoke_TriggerToWorkerExecution(t *testing.T) {
	testDatabaseDSN := strings.TrimSpace(os.Getenv("OCTOMANGER_TEST_DATABASE_DSN"))
	if testDatabaseDSN == "" {
		t.Skip("OCTOMANGER_TEST_DATABASE_DSN is not set")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rootDir := repoRoot(t)
	db := openSchemaPool(t, ctx, testDatabaseDSN)
	defer db.Close()

	if err := migrations.Apply(ctx, db, filepath.Join(rootDir, "db/migrations/*.sql")); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	logger := logging.New(platformconfig.LoggingConfig{
		Level:  "error",
		Format: "text",
	})
	plugins := pluginapp.New(
		fsrepo.New(filepath.Join(rootDir, "plugins/modules")),
		"python3",
		filepath.Join(rootDir, "plugins/sdk/python"),
	)
	jobs := jobapp.New(logger, jobpostgres.New(db), plugins, "test-worker")
	triggers := triggerapp.New(triggerpostgres.New(db), jobs)
	agents := agentapp.New(logger, agentpostgres.New(db), plugins, "test-worker", 50*time.Millisecond, 50*time.Millisecond)
	system := systemapp.New(db, plugins)
	accountTypes := accounttypeapp.New(accounttypepostgres.New(db))
	accounts := accountapp.New(accountpostgres.New(db))
	email := emailapp.New(emailpostgres.New(db))

	mux := http.NewServeMux()
	systemtransport.NewHandler(system).Register(mux)
	plugintransport.NewHandler(plugins).Register(mux)
	jobtransport.NewHandler("", jobs).Register(mux)
	triggertransport.NewHandler("", triggers).Register(mux)
	agenttransport.NewHandler("", agents).Register(mux)
	accounttypestransport.NewHandler("", accountTypes).Register(mux)
	accountstransport.NewHandler("", accounts).Register(mux)
	emailtransport.NewHandler("", email).Register(mux)

	server := httptest.NewServer(mux)
	defer server.Close()

	definition := postJSON[map[string]any](t, server.URL+"/api/v2/job-definitions", map[string]any{
		"key":        "smoke-job",
		"name":       "Smoke Job",
		"plugin_key": "github",
		"action":     "verify_profile",
		"input": map[string]any{
			"username": "octocat",
		},
	})

	definitionID := int64(definition["id"].(float64))

	triggerCreate := postJSON[map[string]any](t, server.URL+"/api/v2/triggers", map[string]any{
		"key":               "smoke-trigger",
		"name":              "Smoke Trigger",
		"job_definition_id": definitionID,
		"mode":              "async",
		"default_input": map[string]any{
			"username": "octocat",
		},
		"enabled": true,
	})

	deliveryToken := triggerCreate["delivery_token"].(string)
	fireResponse := postJSONWithHeaders[map[string]any](t, server.URL+"/api/v2/webhooks/smoke-trigger", map[string]any{
		"username": "hubot",
	}, map[string]string{
		"X-Trigger-Token": deliveryToken,
	})
	if fireResponse["execution_id"] == nil {
		t.Fatalf("expected execution_id in trigger fire response")
	}
	executionID := int64(fireResponse["execution_id"].(float64))

	processed, err := jobs.ProcessNextExecution(ctx)
	if err != nil {
		t.Fatalf("process next execution: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued execution to be processed")
	}

	executionDetails := getJSON[map[string]any](t, fmt.Sprintf("%s/api/v2/job-executions/%d", server.URL, executionID))
	if executionDetails["status"] != "succeeded" {
		t.Fatalf("expected succeeded execution, got %v", executionDetails["status"])
	}

	var logCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM job_logs WHERE job_execution_id = $1`, executionID).Scan(&logCount); err != nil {
		t.Fatalf("count job logs: %v", err)
	}
	if logCount == 0 {
		t.Fatalf("expected imported plugin events to be stored")
	}
}

func postJSON[T any](t *testing.T, url string, payload any) T {
	t.Helper()
	return postJSONWithHeaders[T](t, url, payload, nil)
}

func postJSONWithHeaders[T any](t *testing.T, url string, payload any, headers map[string]string) T {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
	var result T
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func getJSON[T any](t *testing.T, url string) T {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
	var result T
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func openSchemaPool(t *testing.T, ctx context.Context, baseURL string) *pgxpool.Pool {
	t.Helper()

	adminPool, err := pgxpool.New(ctx, baseURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)

	schemaName := fmt.Sprintf("octomanger_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName))
	})

	config, err := pgxpool.ParseConfig(baseURL)
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("open schema pool: %v", err)
	}
	return pool
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}
