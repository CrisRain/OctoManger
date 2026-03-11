package repository

import (
	"context"

	"gorm.io/gorm"
	"octomanger/backend/internal/model"
)

type JobRunRepository interface {
	Create(ctx context.Context, item *model.JobRun) error
	Update(ctx context.Context, item *model.JobRun) error
	GetByID(ctx context.Context, id uint64) (*model.JobRun, error)
	GetLatestByJobID(ctx context.Context, jobID uint64) (*model.JobRun, error)
	ListByJobID(ctx context.Context, jobID uint64, limit, offset int) ([]model.JobRun, int64, error)
	ListByJobTypeKey(ctx context.Context, typeKey string, limit, offset int) ([]model.JobRunWithJob, int64, error)
	ListPaged(ctx context.Context, filter JobRunListFilter, limit, offset int) ([]model.JobRunWithJob, int64, error)
	Delete(ctx context.Context, id uint64) error
}

type jobRunRepository struct {
	db *gorm.DB
}

type JobRunListFilter struct {
	JobID     *uint64
	TypeKey   string
	ActionKey string
	WorkerID  string
	Outcome   string
}

func NewJobRunRepository(db *gorm.DB) JobRunRepository {
	return &jobRunRepository{db: db}
}

func (r *jobRunRepository) Create(ctx context.Context, item *model.JobRun) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *jobRunRepository) Update(ctx context.Context, item *model.JobRun) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *jobRunRepository) GetByID(ctx context.Context, id uint64) (*model.JobRun, error) {
	var item model.JobRun
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *jobRunRepository) GetLatestByJobID(ctx context.Context, jobID uint64) (*model.JobRun, error) {
	var item model.JobRun
	if err := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("started_at DESC").
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *jobRunRepository) ListByJobID(ctx context.Context, jobID uint64, limit, offset int) ([]model.JobRun, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.JobRun{}).Where("job_id = ?", jobID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.JobRun
	err := base.Order("started_at DESC").Limit(limit).Offset(offset).Find(&items).Error
	return items, total, err
}

func (r *jobRunRepository) ListByJobTypeKey(ctx context.Context, typeKey string, limit, offset int) ([]model.JobRunWithJob, int64, error) {
	base := r.runWithJobBase(ctx).Where("jobs.type_key = ?", typeKey)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.JobRunWithJob
	err := base.Select("job_runs.*, jobs.type_key AS job_type_key, jobs.action_key AS job_action_key").
		Order("job_runs.started_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&items).Error
	return items, total, err
}

func (r *jobRunRepository) ListPaged(ctx context.Context, filter JobRunListFilter, limit, offset int) ([]model.JobRunWithJob, int64, error) {
	base := r.runWithJobBase(ctx)

	if filter.JobID != nil && *filter.JobID > 0 {
		base = base.Where("job_runs.job_id = ?", *filter.JobID)
	}
	if filter.TypeKey != "" {
		base = base.Where("jobs.type_key = ?", filter.TypeKey)
	}
	if filter.ActionKey != "" {
		base = base.Where("jobs.action_key = ?", filter.ActionKey)
	}
	if filter.WorkerID != "" {
		base = base.Where("job_runs.worker_id = ?", filter.WorkerID)
	}

	switch filter.Outcome {
	case "success":
		base = base.Where("(job_runs.error_code = '' OR job_runs.error_code IS NULL) AND (job_runs.error_message = '' OR job_runs.error_message IS NULL)")
	case "failed":
		base = base.Where("(job_runs.error_code <> '' AND job_runs.error_code IS NOT NULL) OR (job_runs.error_message <> '' AND job_runs.error_message IS NOT NULL)")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.JobRunWithJob
	err := base.Select("job_runs.*, jobs.type_key AS job_type_key, jobs.action_key AS job_action_key").
		Order("job_runs.started_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&items).Error
	return items, total, err
}

func (r *jobRunRepository) runWithJobBase(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("job_runs").
		Joins("JOIN jobs ON jobs.id = job_runs.job_id")
}

func (r *jobRunRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.JobRun{}, id).Error
}
