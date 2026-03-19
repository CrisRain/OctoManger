package emailpostgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	emaildomain "octomanger/internal/domains/email/domain"
)

var ErrNotFound = errors.New("email account not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]emaildomain.Account, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, address, provider, status, config_json, created_at, updated_at
		FROM email_accounts ORDER BY created_at DESC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list email accounts: %w", err)
	}
	defer rows.Close()

	items := make([]emaildomain.Account, 0)
	for rows.Next() {
		item, err := scanEmail(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) Get(ctx context.Context, emailID int64) (*emaildomain.Account, error) {
	row := r.db.WithContext(ctx).Raw(`
		SELECT id, address, provider, status, config_json, created_at, updated_at
		FROM email_accounts WHERE id = $1`, emailID).Row()
	item, err := scanEmail(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r Repository) Create(ctx context.Context, input emaildomain.CreateInput) (*emaildomain.Account, error) {
	configJSON, err := json.Marshal(normalizeMap(input.Config))
	if err != nil {
		return nil, fmt.Errorf("marshal email config: %w", err)
	}

	var emailID int64
	row := r.db.WithContext(ctx).Raw(`
		INSERT INTO email_accounts (address, provider, status, config_json)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		input.Address, input.Provider, input.Status, configJSON,
	).Row()
	if err := row.Scan(&emailID); err != nil {
		return nil, fmt.Errorf("create email account: %w", err)
	}

	return r.Get(ctx, emailID)
}

func (r Repository) Patch(ctx context.Context, emailID int64, input emaildomain.PatchInput) (*emaildomain.Account, error) {
	current, err := r.Get(ctx, emailID)
	if err != nil {
		return nil, err
	}

	if input.Provider != nil {
		current.Provider = *input.Provider
	}
	if input.Status != nil {
		current.Status = *input.Status
	}
	if input.Config != nil {
		current.Config = input.Config
	}

	configJSON, err := json.Marshal(normalizeMap(current.Config))
	if err != nil {
		return nil, fmt.Errorf("marshal email config: %w", err)
	}

	result := r.db.WithContext(ctx).Exec(`
		UPDATE email_accounts
		SET provider = $2, status = $3, config_json = $4, updated_at = NOW()
		WHERE id = $1`,
		emailID, current.Provider, current.Status, configJSON,
	)
	if result.Error != nil {
		return nil, fmt.Errorf("patch email account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.Get(ctx, emailID)
}

func (r Repository) Delete(ctx context.Context, emailID int64) error {
	result := r.db.WithContext(ctx).Exec(`DELETE FROM email_accounts WHERE id = $1`, emailID)
	if result.Error != nil {
		return fmt.Errorf("delete email account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEmail(row scanner) (emaildomain.Account, error) {
	var item emaildomain.Account
	var configJSON []byte
	if err := row.Scan(
		&item.ID, &item.Address, &item.Provider, &item.Status,
		&configJSON, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return emaildomain.Account{}, ErrNotFound
		}
		return emaildomain.Account{}, fmt.Errorf("scan email account: %w", err)
	}
	item.Config = decodeJSONMap(configJSON)
	return item, nil
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	v := map[string]any{}
	_ = json.Unmarshal(raw, &v)
	return v
}

func normalizeMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
