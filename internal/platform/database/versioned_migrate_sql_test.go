package database

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func expectEnsureSchemaMigrationTable(mock sqlmock.Sqlmock) {
	mock.ExpectExec("(?s).*CREATE TABLE IF NOT EXISTS schema_migrations.*").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("(?s).*ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS name TEXT.*").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("(?s).*ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS applied_at.*").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("(?s).*UPDATE schema_migrations SET name = COALESCE.*").WillReturnResult(sqlmock.NewResult(0, 0))
}

func expectSchemaMigrationVersionType(mock sqlmock.Sqlmock) {
	rows := sqlmock.NewRows([]string{"data_type"}).AddRow("bigint")
	mock.ExpectQuery("(?is).*SELECT data_type.*information_schema.columns.*").WillReturnRows(rows)
}

func TestEnsureSchemaMigrationTable(t *testing.T) {
	db, mock := newSQLMockDB(t)
	expectEnsureSchemaMigrationTable(mock)

	if err := ensureSchemaMigrationTable(context.Background(), db); err != nil {
		t.Fatalf("ensure schema migration table: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestApplyMigration(t *testing.T) {
	db, mock := newSQLMockDB(t)

	mock.ExpectBegin()
	expectSchemaMigrationVersionType(mock)
	mock.ExpectExec("(?s).*CREATE TABLE test_table.*").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("(?s).*INSERT.*schema_migrations.*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := applyMigration(context.Background(), db, versionedMigration{
		Version: 1,
		Name:    "test",
		Up:      []string{"CREATE TABLE test_table (id INT)"},
	})
	if err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRollbackLastVersionedMigrationNoRows(t *testing.T) {
	db, mock := newSQLMockDB(t)
	expectEnsureSchemaMigrationTable(mock)
	expectSchemaMigrationVersionType(mock)
	mock.ExpectQuery("(?is).*FROM.*schema_migrations.*ORDER BY.*version.*LIMIT.*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name"}))

	if err := rollbackLastVersionedMigrationWith(context.Background(), db, nil); err != nil {
		t.Fatalf("expected nil when no rows, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRollbackLastVersionedMigrationMissingDefinition(t *testing.T) {
	db, mock := newSQLMockDB(t)
	expectEnsureSchemaMigrationTable(mock)
	expectSchemaMigrationVersionType(mock)
	mock.ExpectQuery("(?is).*FROM.*schema_migrations.*ORDER BY.*version.*LIMIT.*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name"}).AddRow(int64(99), "missing"))

	err := rollbackLastVersionedMigrationWith(context.Background(), db, []versionedMigration{{Version: 1, Name: "known"}})
	if err == nil {
		t.Fatalf("expected error for missing migration")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRollbackLastVersionedMigrationSuccess(t *testing.T) {
	db, mock := newSQLMockDB(t)
	expectEnsureSchemaMigrationTable(mock)
	expectSchemaMigrationVersionType(mock)
	mock.ExpectQuery("(?is).*FROM.*schema_migrations.*ORDER BY.*version.*LIMIT.*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name"}).AddRow(int64(1), "initial"))

	mock.ExpectBegin()
	mock.ExpectExec("(?s).*DROP TABLE test_table.*").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("(?s).*DELETE FROM.*schema_migrations.*").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := rollbackLastVersionedMigrationWith(context.Background(), db, []versionedMigration{{
		Version: 1,
		Name:    "initial",
		Down:    []string{"DROP TABLE test_table"},
	}})
	if err != nil {
		t.Fatalf("rollback migration: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
