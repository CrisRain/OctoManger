package runtime

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"octomanger/internal/domains/plugins/grpcclient"
	"octomanger/internal/testutil"
)

func TestAppCloseInvokesHooks(t *testing.T) {
	app := &App{
		Logger:  zap.NewNop(),
		DB:      testutil.NewTestDB(t),
		Redis:   func() *redis.Client { c, _ := testutil.NewTestRedis(t); return c }(),
		Plugins: grpcclient.New(grpcclient.NewStaticRegistry(map[string]grpcclient.PluginServiceConfig{})),
	}

	flushed := false
	app.flushLogs = func() { flushed = true }

	origCloseSQL := closeSQLDB
	origCloseRedis := closeRedis
	defer func() {
		closeSQLDB = origCloseSQL
		closeRedis = origCloseRedis
	}()

	sawSQL := false
	sawRedis := false
	closeSQLDB = func(db *sql.DB) error {
		sawSQL = true
		return errors.New("close db")
	}
	closeRedis = func(rdb *redis.Client) error {
		sawRedis = true
		return errors.New("close redis")
	}

	app.Close()

	if !flushed {
		t.Fatalf("expected flushLogs to be called")
	}
	if !sawSQL || !sawRedis {
		t.Fatalf("expected close hooks to run")
	}
}
