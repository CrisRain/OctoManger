package systemapp

import (
	"context"
	"time"

	"gorm.io/gorm"

	pluginapp "octomanger/internal/domains/plugins/app"
)

type Status struct {
	Now         time.Time `json:"now"`
	DatabaseOK  bool      `json:"database_ok"`
	PluginCount int       `json:"plugin_count"`
}

// DashboardSummary is returned by GET /api/v2/dashboard and replaces 8 parallel
// frontend requests with a single round-trip. Counts are produced by cheap SQL
// COUNT queries; recent executions fetch the latest 6 rows only.
type DashboardSummary struct {
	PluginCount        int              `json:"pluginCount"`
	AccountTypeCount   int              `json:"accountTypeCount"`
	AccountCount       int              `json:"accountCount"`
	EmailAccountCount  int              `json:"emailAccountCount"`
	JobDefinitionCount int              `json:"jobDefinitionCount"`
	JobExecutionCount  int              `json:"jobExecutionCount"`
	TriggerCount       int              `json:"triggerCount"`
	AgentCount         int              `json:"agentCount"`
	RecentExecutions   []RecentExecution `json:"recentExecutions"`
}

// RecentExecution is a lightweight projection of job_executions for the
// dashboard table. It mirrors the fields the frontend currently reads.
type RecentExecution struct {
	ID             int64  `json:"id"`
	DefinitionName string `json:"definition_name"`
	PluginKey      string `json:"plugin_key"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	WorkerID       string `json:"worker_id"`
}

type Service struct {
	db      *gorm.DB
	plugins pluginapp.Service
}

func New(db *gorm.DB, plugins pluginapp.Service) Service {
	return Service{db: db, plugins: plugins}
}

func (s Service) Status(ctx context.Context) (Status, error) {
	plugins, err := s.plugins.List(ctx)
	if err != nil {
		return Status{}, err
	}

	return Status{
		Now:         time.Now().UTC(),
		DatabaseOK:  s.db != nil,
		PluginCount: len(plugins),
	}, nil
}

// DashboardSummary returns entity counts and the 6 most recent executions in a
// single DB round-trip (one query per table — all cheap COUNT/LIMIT queries).
func (s Service) DashboardSummary(ctx context.Context) (DashboardSummary, error) {
	type countRow struct{ N int64 }

	countOf := func(table string) (int, error) {
		var row countRow
		if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) AS n FROM "+table).Scan(&row).Error; err != nil {
			return 0, err
		}
		return int(row.N), nil
	}

	pluginList, _ := s.plugins.List(ctx) // plugins live in-memory; no separate table

	accountTypeCount, err := countOf("account_types")
	if err != nil { return DashboardSummary{}, err }

	accountCount, err := countOf("accounts")
	if err != nil { return DashboardSummary{}, err }

	emailAccountCount, err := countOf("email_accounts")
	if err != nil { return DashboardSummary{}, err }

	jobDefinitionCount, err := countOf("job_definitions")
	if err != nil { return DashboardSummary{}, err }

	jobExecutionCount, err := countOf("job_executions")
	if err != nil { return DashboardSummary{}, err }

	triggerCount, err := countOf("triggers")
	if err != nil { return DashboardSummary{}, err }

	agentCount, err := countOf("agents")
	if err != nil { return DashboardSummary{}, err }

	// Fetch the 6 most recent executions — a lightweight projection, not the full scan.
	rows, err := s.db.WithContext(ctx).Raw(`
		SELECT e.id, d.name, d.plugin_key, d.action, e.status, COALESCE(e.worker_id, '')
		FROM job_executions e
		JOIN job_definitions d ON d.id = e.job_definition_id
		ORDER BY e.created_at DESC
		LIMIT 6`).Rows()
	if err != nil {
		return DashboardSummary{}, err
	}
	defer rows.Close()

	recent := make([]RecentExecution, 0, 6)
	for rows.Next() {
		var r RecentExecution
		if err := rows.Scan(&r.ID, &r.DefinitionName, &r.PluginKey, &r.Action, &r.Status, &r.WorkerID); err != nil {
			return DashboardSummary{}, err
		}
		recent = append(recent, r)
	}
	if err := rows.Err(); err != nil {
		return DashboardSummary{}, err
	}

	return DashboardSummary{
		PluginCount:        len(pluginList),
		AccountTypeCount:   accountTypeCount,
		AccountCount:       accountCount,
		EmailAccountCount:  emailAccountCount,
		JobDefinitionCount: jobDefinitionCount,
		JobExecutionCount:  jobExecutionCount,
		TriggerCount:       triggerCount,
		AgentCount:         agentCount,
		RecentExecutions:   recent,
	}, nil
}
