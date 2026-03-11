package repository

import (
	"context"

	"gorm.io/gorm"
	"octomanger/backend/internal/model"
)

type JobRepository interface {
	List(ctx context.Context) ([]model.Job, error)
	ListPaged(ctx context.Context, limit, offset int) ([]model.Job, int64, error)
	GetByID(ctx context.Context, id uint64) (*model.Job, error)
	Create(ctx context.Context, item *model.Job) error
	Update(ctx context.Context, item *model.Job) error
	UpdateStatus(ctx context.Context, id uint64, status int16) (*model.Job, error)
	Summary(ctx context.Context) (JobStatusSummary, error)
	Delete(ctx context.Context, id uint64) error
}

type jobRepository struct {
	db *gorm.DB
}

type JobStatusSummary struct {
	Total    int64
	Queued   int64
	Running  int64
	Done     int64
	Failed   int64
	Canceled int64
}

func NewJobRepository(db *gorm.DB) JobRepository {
	return &jobRepository{db: db}
}

func (r *jobRepository) List(ctx context.Context) ([]model.Job, error) {
	var items []model.Job
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *jobRepository) ListPaged(ctx context.Context, limit, offset int) ([]model.Job, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.Job{})

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.Job
	err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error
	return items, total, err
}

func (r *jobRepository) GetByID(ctx context.Context, id uint64) (*model.Job, error) {
	var item model.Job
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *jobRepository) Create(ctx context.Context, item *model.Job) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *jobRepository) Update(ctx context.Context, item *model.Job) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *jobRepository) UpdateStatus(ctx context.Context, id uint64, status int16) (*model.Job, error) {
	if err := r.db.WithContext(ctx).
		Model(&model.Job{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *jobRepository) Summary(ctx context.Context) (JobStatusSummary, error) {
	rows := []struct {
		Status int16
		Total  int64
	}{}

	if err := r.db.WithContext(ctx).
		Model(&model.Job{}).
		Select("status, COUNT(*) AS total").
		Group("status").
		Scan(&rows).Error; err != nil {
		return JobStatusSummary{}, err
	}

	summary := JobStatusSummary{}
	for _, row := range rows {
		summary.Total += row.Total
		switch row.Status {
		case 0:
			summary.Queued += row.Total
		case 1:
			summary.Running += row.Total
		case 2:
			summary.Done += row.Total
		case 3:
			summary.Failed += row.Total
		case 4:
			summary.Canceled += row.Total
		}
	}

	return summary, nil
}

func (r *jobRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Job{}, id).Error
}
