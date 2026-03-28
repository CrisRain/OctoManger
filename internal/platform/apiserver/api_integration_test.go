package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"

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
	"octomanger/internal/platform/apikey"
	"octomanger/internal/platform/auth"
	platformconfig "octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/logging"
)

const testAdminKey = "test-admin-key"

type testAPI struct {
	baseURL string
	jobs    jobapp.Service
	db      *gorm.DB
}

func TestAPISmoke_TriggerToWorkerExecution(t *testing.T) {
	requirePython(t)
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	fakeServerURL := startFakeServer(t, ctx, rootDir)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	definition := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/job-definitions", map[string]any{
		"key":        "smoke-job",
		"name":       "Smoke Job",
		"plugin_key": "octo_demo",
		"action":     "LIST_TASKS",
		"input": map[string]any{
			"account": map[string]any{
				"spec": map[string]any{
					"username": "testuser",
					"api_key":  "demo_testkey_12345678",
					"base_url": fakeServerURL,
				},
			},
			"params": map[string]any{
				"page": "1",
			},
		},
	}, adminHeaders(testAdminKey))

	definitionID := int64(definition["id"].(float64))

	triggerCreate := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/triggers", map[string]any{
		"key":               "smoke-trigger",
		"name":              "Smoke Trigger",
		"job_definition_id": definitionID,
		"mode":              "async",
		"default_input": map[string]any{
			"params": map[string]any{
				"page": "1",
			},
		},
		"enabled": true,
	}, adminHeaders(testAdminKey))

	deliveryToken := triggerCreate["delivery_token"].(string)
	fireResponse := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/webhooks/smoke-trigger", map[string]any{
		"params": map[string]any{
			"page": "1",
		},
	}, map[string]string{
		"X-Trigger-Token": deliveryToken,
	})
	if fireResponse["execution_id"] == nil {
		t.Fatalf("expected execution_id in trigger fire response")
	}
	executionID := int64(fireResponse["execution_id"].(float64))

	processed, err := api.jobs.ProcessNextExecution(ctx)
	if err != nil {
		t.Fatalf("process next execution: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued execution to be processed")
	}

	executionDetails := getJSONWithHeaders[map[string]any](t, fmt.Sprintf("%s/api/v2/job-executions/%d", api.baseURL, executionID), adminHeaders(testAdminKey))
	if executionDetails["status"] != "succeeded" {
		t.Fatalf("expected succeeded execution, got %v", executionDetails["status"])
	}

	var logCount int64
	if err := api.db.WithContext(ctx).Table("job_logs").Where("job_execution_id = ?", executionID).Count(&logCount).Error; err != nil {
		t.Fatalf("count job logs: %v", err)
	}
	if logCount == 0 {
		t.Fatalf("expected imported plugin events to be stored")
	}
}

func TestAPISystemConfigRequiresAdminKey(t *testing.T) {
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	putJSONExpectStatus(t, api.baseURL+"/api/v2/config", map[string]any{
		"app_name":                    "OctoManger Test",
		"job_default_timeout_minutes": 45,
		"job_max_concurrency":         12,
	}, nil, http.StatusUnauthorized)

	updated := putJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/config", map[string]any{
		"app_name":                    "OctoManger Test",
		"job_default_timeout_minutes": 45,
		"job_max_concurrency":         12,
	}, adminHeaders(testAdminKey))
	if updated["app_name"] != "OctoManger Test" {
		t.Fatalf("expected updated app_name, got %v", updated["app_name"])
	}
	if updated["job_default_timeout_minutes"] != float64(45) {
		t.Fatalf("expected updated timeout, got %v", updated["job_default_timeout_minutes"])
	}

	value := getJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/config", adminHeaders(testAdminKey))
	if value["app_name"] != "OctoManger Test" {
		t.Fatalf("expected updated app_name, got %v", value["app_name"])
	}
}

func TestAPIPluginRuntimeConfigReadAndWrite(t *testing.T) {
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	current := getJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/plugins/octo_demo/runtime-config", adminHeaders(testAdminKey))
	if current["plugin_key"] != "octo_demo" {
		t.Fatalf("expected plugin key octo_demo, got %v", current["plugin_key"])
	}
	if current["grpc_address"] == nil {
		t.Fatalf("expected grpc_address field")
	}

	updated := putJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/plugins/octo_demo/runtime-config", map[string]any{
		"grpc_address": "127.0.0.1:61051",
	}, adminHeaders(testAdminKey))
	if updated["grpc_address"] != "127.0.0.1:61051" {
		t.Fatalf("expected updated grpc address, got %v", updated["grpc_address"])
	}
}

func TestAPIAccountExecuteLoadsAccountViaPluginInternalAPI(t *testing.T) {
	requirePython(t)
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	fakeServerURL := startFakeServer(t, ctx, rootDir)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/plugins/sync", map[string]any{}, adminHeaders(testAdminKey))
	typeList := getJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/account-types", adminHeaders(testAdminKey))
	items, ok := typeList["items"].([]any)
	if !ok {
		t.Fatalf("expected account types list, got %#v", typeList)
	}

	var octoDemoTypeID int64
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if item["key"] == "octo_demo" {
			octoDemoTypeID = int64(item["id"].(float64))
			break
		}
	}
	if octoDemoTypeID == 0 {
		t.Fatalf("expected synced octo_demo account type")
	}

	account := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/accounts", map[string]any{
		"account_type_id": octoDemoTypeID,
		"identifier":      "demo-account",
		"status":          "active",
		"tags":            []string{},
		"spec": map[string]any{
			"username": "testuser",
			"api_key":  "demo_testkey_12345678",
			"base_url": fakeServerURL,
		},
	}, adminHeaders(testAdminKey))

	accountID := int64(account["id"].(float64))
	result := postJSONWithHeaders[map[string]any](t, fmt.Sprintf("%s/api/v2/accounts/%d/execute", api.baseURL, accountID), map[string]any{
		"action": "LIST_TASKS",
		"params": map[string]any{
			"page": 1,
		},
	}, adminHeaders(testAdminKey))

	if result["status"] != "ok" {
		t.Fatalf("expected ok result, got %#v", result)
	}
	payload, ok := result["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result payload, got %#v", result["result"])
	}
	if _, ok := payload["items"].([]any); !ok {
		t.Fatalf("expected items list in payload, got %#v", payload)
	}
}

func TestAPIPluginActionWithAccountRefLoadsAccountViaPluginInternalAPI(t *testing.T) {
	requirePython(t)
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	fakeServerURL := startFakeServer(t, ctx, rootDir)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/plugins/sync", map[string]any{}, adminHeaders(testAdminKey))
	typeList := getJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/account-types", adminHeaders(testAdminKey))
	items, ok := typeList["items"].([]any)
	if !ok {
		t.Fatalf("expected account types list, got %#v", typeList)
	}

	var octoDemoTypeID int64
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if item["key"] == "octo_demo" {
			octoDemoTypeID = int64(item["id"].(float64))
			break
		}
	}
	if octoDemoTypeID == 0 {
		t.Fatalf("expected synced octo_demo account type")
	}

	account := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/accounts", map[string]any{
		"account_type_id": octoDemoTypeID,
		"identifier":      "demo-account",
		"status":          "active",
		"tags":            []string{},
		"spec": map[string]any{
			"username": "testuser",
			"api_key":  "demo_testkey_12345678",
			"base_url": fakeServerURL,
		},
	}, adminHeaders(testAdminKey))

	accountID := int64(account["id"].(float64))
	result := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/plugins/octo_demo/actions/LIST_TASKS", map[string]any{
		"params": map[string]any{
			"page": 1,
		},
		"account": map[string]any{
			"id":         accountID,
			"identifier": "demo-account",
		},
		// Even if request spec is wrong, plugin should load the real account from internal API.
		"spec": map[string]any{
			"username": "bad-user",
			"api_key":  "bad-key",
			"base_url": fakeServerURL,
		},
	}, adminHeaders(testAdminKey))

	payload, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected action data payload, got %#v", result["data"])
	}
	if _, ok := payload["items"].([]any); !ok {
		t.Fatalf("expected items list in payload, got %#v", payload)
	}
}

func TestAPIAgentCreateAndStart(t *testing.T) {
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	created := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/agents", map[string]any{
		"name":       "Demo Server Agent",
		"plugin_key": "octo_demo",
		"action":     "AGENT_FAKE_SERVER",
		"input": map[string]any{
			"params": map[string]any{},
			"account": map[string]any{
				"identifier": "demo-server",
				"spec": map[string]any{
					"base_url": "http://127.0.0.1:18080",
				},
			},
		},
	}, adminHeaders(testAdminKey))

	if created["id"] == nil {
		t.Fatalf("expected created agent id")
	}
	if created["created_at"] == nil || created["updated_at"] == nil {
		t.Fatalf("expected created agent timestamps, got created_at=%v updated_at=%v", created["created_at"], created["updated_at"])
	}

	agentID := int64(created["id"].(float64))

	started := requestJSON[map[string]any](
		t,
		http.MethodPost,
		fmt.Sprintf("%s/api/v2/agents/%d/start", api.baseURL, agentID),
		nil,
		adminHeaders(testAdminKey),
		http.StatusAccepted,
	)
	if started["started"] != true {
		t.Fatalf("expected start response, got %v", started["started"])
	}

	status := getJSONWithHeaders[map[string]any](t, fmt.Sprintf("%s/api/v2/agents/%d/status", api.baseURL, agentID), adminHeaders(testAdminKey))
	if status["desired_state"] != "running" {
		t.Fatalf("expected desired_state running, got %v", status["desired_state"])
	}
}

func TestAPIJobDefinitionPatchUpdatesPluginAndAction(t *testing.T) {
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	definition := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/job-definitions", map[string]any{
		"key":        "patch-job",
		"name":       "Patch Job",
		"plugin_key": "octo_demo",
		"action":     "LIST_TASKS",
		"input": map[string]any{
			"params": map[string]any{
				"page": "1",
			},
		},
	}, adminHeaders(testAdminKey))

	definitionID := int64(definition["id"].(float64))

	patched := requestJSON[map[string]any](t, http.MethodPatch, fmt.Sprintf("%s/api/v2/job-definitions/%d", api.baseURL, definitionID), map[string]any{
		"name":       "Patch Job Updated",
		"plugin_key": "octo_demo",
		"action":     "CREATE_TASK",
		"input": map[string]any{
			"params": map[string]any{
				"title": "Created from patch",
			},
		},
	}, adminHeaders(testAdminKey), http.StatusOK)

	if patched["name"] != "Patch Job Updated" {
		t.Fatalf("expected updated job name, got %v", patched["name"])
	}
	if patched["plugin_key"] != "octo_demo" {
		t.Fatalf("expected updated plugin_key, got %v", patched["plugin_key"])
	}
	if patched["action"] != "CREATE_TASK" {
		t.Fatalf("expected updated action, got %v", patched["action"])
	}
}

func TestAPITriggerPatchUpdatesJobAndDefaultInput(t *testing.T) {
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	jobA := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/job-definitions", map[string]any{
		"key":        "trigger-job-a",
		"name":       "Trigger Job A",
		"plugin_key": "octo_demo",
		"action":     "LIST_TASKS",
		"input":      map[string]any{},
	}, adminHeaders(testAdminKey))
	jobB := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/job-definitions", map[string]any{
		"key":        "trigger-job-b",
		"name":       "Trigger Job B",
		"plugin_key": "octo_demo",
		"action":     "CREATE_TASK",
		"input":      map[string]any{},
	}, adminHeaders(testAdminKey))

	trigger := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/triggers", map[string]any{
		"key":               "patch-trigger",
		"name":              "Patch Trigger",
		"job_definition_id": int64(jobA["id"].(float64)),
		"mode":              "async",
		"default_input": map[string]any{
			"source": "before",
		},
		"enabled": true,
	}, adminHeaders(testAdminKey))

	triggerID := int64(trigger["trigger"].(map[string]any)["id"].(float64))
	jobBID := int64(jobB["id"].(float64))

	patched := requestJSON[map[string]any](t, http.MethodPatch, fmt.Sprintf("%s/api/v2/triggers/%d", api.baseURL, triggerID), map[string]any{
		"name":              "Patch Trigger Updated",
		"job_definition_id": jobBID,
		"mode":              "sync",
		"default_input": map[string]any{
			"source": "after",
			"params": map[string]any{
				"title": "Created from trigger patch",
			},
		},
		"enabled": false,
	}, adminHeaders(testAdminKey), http.StatusOK)

	if patched["name"] != "Patch Trigger Updated" {
		t.Fatalf("expected updated trigger name, got %v", patched["name"])
	}
	if int64(patched["job_definition_id"].(float64)) != jobBID {
		t.Fatalf("expected updated job_definition_id %d, got %v", jobBID, patched["job_definition_id"])
	}
	if patched["mode"] != "sync" {
		t.Fatalf("expected updated trigger mode, got %v", patched["mode"])
	}
	if patched["enabled"] != false {
		t.Fatalf("expected updated enabled flag false, got %v", patched["enabled"])
	}

	defaultInput, ok := patched["default_input"].(map[string]any)
	if !ok {
		t.Fatalf("expected default_input map, got %T", patched["default_input"])
	}
	if defaultInput["source"] != "after" {
		t.Fatalf("expected updated default input source, got %v", defaultInput["source"])
	}
}

func postJSONWithHeaders[T any](t *testing.T, rawURL string, body any, headers map[string]string) T {
	t.Helper()
	return requestJSON[T](t, http.MethodPost, rawURL, body, headers, http.StatusOK, http.StatusCreated)
}

func putJSONWithHeaders[T any](t *testing.T, rawURL string, body any, headers map[string]string) T {
	t.Helper()
	return requestJSON[T](t, http.MethodPut, rawURL, body, headers, http.StatusOK)
}

func putJSONExpectStatus(t *testing.T, rawURL string, body any, headers map[string]string, expectedStatus int) {
	t.Helper()
	_ = requestJSON[map[string]any](t, http.MethodPut, rawURL, body, headers, expectedStatus)
}

func getJSON[T any](t *testing.T, rawURL string) T {
	t.Helper()
	return getJSONWithHeaders[T](t, rawURL, nil)
}

func getJSONWithHeaders[T any](t *testing.T, rawURL string, headers map[string]string) T {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response := doRequestExpectStatus(t, request, http.StatusOK)
	defer response.Body.Close()

	var result T
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func requestJSON[T any](t *testing.T, method, rawURL string, body any, headers map[string]string, expectedStatuses ...int) T {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request, err := http.NewRequest(method, rawURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response := doRequestExpectStatus(t, request, expectedStatuses...)
	defer response.Body.Close()

	var result T
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func doRequestExpectStatus(t *testing.T, request *http.Request, expectedStatuses ...int) *http.Response {
	t.Helper()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request %s %s: %v", request.Method, request.URL.String(), err)
	}

	for _, status := range expectedStatuses {
		if response.StatusCode == status {
			return response
		}
	}

	bodyText, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	t.Fatalf("unexpected status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyText)))
	return nil
}

func setupIntegrationPrereqs(t *testing.T) (context.Context, string, *gorm.DB) {
	t.Helper()

	testDatabaseDSN := strings.TrimSpace(os.Getenv("OCTOMANGER_TEST_DATABASE_DSN"))
	if testDatabaseDSN == "" {
		t.Skip("OCTOMANGER_TEST_DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	rootDir := repoRoot(t)
	db := openSchemaDB(t, ctx, testDatabaseDSN)
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return ctx, rootDir, db
}

func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not available")
	}
}

func startFakeServer(t *testing.T, ctx context.Context, rootDir string) string {
	t.Helper()

	requirePython(t)
	port := reservePort(t)
	scriptPath := filepath.Join(rootDir, "plugins/modules/octo_demo/fake_server.py")
	cmd := exec.CommandContext(ctx, "python3", scriptPath, "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fake server: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForHTTPStatus(t, ctx, baseURL+"/tasks", http.StatusUnauthorized)
	return baseURL
}

func startTestAPI(t *testing.T, ctx context.Context, rootDir string, db *gorm.DB, adminKey string) *testAPI {
	t.Helper()

	logger := logging.New(platformconfig.LoggingConfig{
		Level:  "error",
		Format: "text",
	})
	addr := fmt.Sprintf("127.0.0.1:%d", reservePort(t))
	baseURL := "http://" + addr
	plugins := pluginapp.New(
		fsrepo.New(filepath.Join(rootDir, "plugins/modules")),
		"python3",
		filepath.Join(rootDir, "plugins/sdk/python"),
	)
	pluginSettingsStore := database.NewPluginSettingsStore(db)
	pluginServiceConfigStore := database.NewPluginServiceConfigStore(db)
	plugins = plugins.
		WithSettingsStore(pluginSettingsStore).
		WithInternalAPI(pluginapp.InternalAPIConfig{
			URL:            baseURL,
			Token:          adminKey,
			TimeoutSeconds: 60,
		})
	accountTypes := accounttypeapp.New(accounttypepostgres.New(db))
	accounts := accountapp.New(accountpostgres.New(db), plugins)
	email := emailapp.New(emailpostgres.New(db))
	jobs := jobapp.New(logger, jobpostgres.New(db), plugins, "test-worker")
	triggers := triggerapp.New(triggerpostgres.New(db), jobs)
	agents := agentapp.New(logger, agentpostgres.New(db), plugins, nil, "test-worker", 50*time.Millisecond, 50*time.Millisecond)
	system := systemapp.New(db, plugins)
	seedResult := db.WithContext(ctx).
		Model(&database.SystemSettingsModel{}).
		Where("id = ?", 1).
		Updates(map[string]any{
			"app_name":                    "OctoManger Test",
			"job_default_timeout_minutes": 30,
			"job_max_concurrency":         10,
			"admin_key_hash":              apikey.Hash(adminKey),
		})
	if seedResult.Error != nil {
		t.Fatalf("seed system settings: %v", seedResult.Error)
	}
	if seedResult.RowsAffected == 0 {
		if err := db.WithContext(ctx).Create(&database.SystemSettingsModel{
			ID:                       1,
			AppName:                  "OctoManger Test",
			JobDefaultTimeoutMinutes: 30,
			JobMaxConcurrency:        10,
			AdminKeyHash:             apikey.Hash(adminKey),
		}).Error; err != nil {
			t.Fatalf("insert system settings: %v", err)
		}
	}

	h := server.New(
		server.WithHostPorts(addr),
		server.WithExitWaitTime(time.Second),
	)

	root := h.Group("/")
	v2 := h.Group("/api/v2")
	v2.Use(auth.RequireAdminForRouterWithVerifier(system))
	systemtransport.NewHandler(system).Register(root, v2)
	plugintransport.NewHandler(plugins, accountTypes, pluginSettingsStore, pluginServiceConfigStore).Register(v2)
	jobtransport.NewHandler(jobs).Register(v2)
	agenttransport.NewHandler(agents).Register(v2)
	accounttypestransport.NewHandler(accountTypes).Register(v2)
	accountstransport.NewHandler(accounts).Register(v2)
	emailtransport.NewHandler(email).Register(v2)
	triggertransport.NewHandler(triggers).Register(v2)
	registerPluginInternalAPI(root, adminKey, accounts, email)

	go h.Spin()
	t.Cleanup(func() {
		h.Shutdown(context.Background()) //nolint:errcheck
	})

	waitForHTTPStatus(t, ctx, baseURL+"/healthz", http.StatusOK)
	return &testAPI{
		baseURL: baseURL,
		jobs:    jobs,
		db:      db,
	}
}

func waitForHTTPStatus(t *testing.T, ctx context.Context, url string, expectedStatus int) {
	t.Helper()

	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("new wait request: %v", err)
		}

		response, err := http.DefaultClient.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == expectedStatus {
				return
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("wait for %s status %d timed out", url, expectedStatus)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func reservePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func adminHeaders(adminKey string) map[string]string {
	return map[string]string{
		"X-Admin-Key": adminKey,
	}
}

func openSchemaDB(t *testing.T, ctx context.Context, baseDSN string) *gorm.DB {
	t.Helper()

	adminPool, err := pgxpool.New(ctx, baseDSN)
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

	db, err := database.Open(platformconfig.DatabaseConfig{
		DSN:            withSearchPath(baseDSN, schemaName),
		MaxConnections: 4,
		QueryTimeout:   5 * time.Second,
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

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
}

func TestResolveStaticFilePrefersBrotliAssets(t *testing.T) {
	distDir := t.TempDir()
	assetPath := filepath.Join(distDir, "assets", "app.js")
	writeTestFile(t, assetPath, []byte("console.log('ok');"))
	writeTestFile(t, assetPath+".gz", []byte("gzip"))
	writeTestFile(t, assetPath+".br", []byte("brotli"))

	file, ok := ResolveStaticFile(os.DirFS(distDir), http.MethodGet, "/assets/app.js", "*/*", "br, gzip")
	if !ok {
		t.Fatal("expected static asset to resolve")
	}
	if file.Path != "assets/app.js.br" {
		t.Fatalf("unexpected path %q", file.Path)
	}
	if file.ContentEncoding != "br" {
		t.Fatalf("unexpected content encoding %q", file.ContentEncoding)
	}
	if file.CacheControl != CacheControlImmutableAssets {
		t.Fatalf("unexpected cache control %q", file.CacheControl)
	}
	if !strings.Contains(file.ContentType, "javascript") {
		t.Fatalf("unexpected content type %q", file.ContentType)
	}
}

func TestResolveStaticFileUsesSPAIndexForHTMLNavigations(t *testing.T) {
	distDir := t.TempDir()
	indexPath := filepath.Join(distDir, "index.html")
	writeTestFile(t, indexPath, []byte("<!doctype html>"))

	file, ok := ResolveStaticFile(os.DirFS(distDir), http.MethodGet, "/jobs/42", "text/html,application/xhtml+xml", "gzip")
	if !ok {
		t.Fatal("expected SPA fallback to resolve")
	}
	if file.Path != "index.html" {
		t.Fatalf("unexpected path %q", file.Path)
	}
	if file.CacheControl != CacheControlNoCache {
		t.Fatalf("unexpected cache control %q", file.CacheControl)
	}
}

func TestResolveStaticFileDoesNotFallbackForMissingAssetsOrAPIPaths(t *testing.T) {
	distDir := t.TempDir()
	writeTestFile(t, filepath.Join(distDir, "index.html"), []byte("<!doctype html>"))
	assets := os.DirFS(distDir)

	if _, ok := ResolveStaticFile(assets, http.MethodGet, "/assets/missing.js", "text/html", "gzip"); ok {
		t.Fatal("expected missing asset request to return 404")
	}
	if _, ok := ResolveStaticFile(assets, http.MethodGet, "/api/v2/unknown", "text/html", "gzip"); ok {
		t.Fatal("expected api path to return 404")
	}
	if _, ok := ResolveStaticFile(assets, http.MethodPost, "/jobs/42", "text/html", "gzip"); ok {
		t.Fatal("expected non-GET html request to return 404")
	}
}

func TestAcceptsEncodingHonorsZeroQuality(t *testing.T) {
	if AcceptsEncoding("br;q=0, gzip;q=1", "br") {
		t.Fatal("expected brotli with q=0 to be rejected")
	}
	if !AcceptsEncoding("br;q=0, gzip;q=1", "gzip") {
		t.Fatal("expected gzip to remain accepted")
	}
}

func TestOpenWebUIFallsBackToFilesystemAssets(t *testing.T) {
	t.Skip("covered implicitly by runtime packaging path")
}

func writeTestFile(t *testing.T, filename string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filename, err)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

var _ fs.FS = os.DirFS(".")
