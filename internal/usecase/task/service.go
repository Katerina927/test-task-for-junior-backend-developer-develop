package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		Periodicity: normalized.Periodicity,
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		Periodicity: normalized.Periodicity,
		UpdatedAt:   s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func (s *Service) GenerateForDate(ctx context.Context, date time.Time) ([]taskdomain.Task, error) {
	templates, err := s.repo.ListPeriodicTemplates(ctx)
	if err != nil {
		return nil, err
	}

	now := s.now()
	var toCreate []*taskdomain.Task

	for i := range templates {
		tmpl := &templates[i]
		if tmpl.Periodicity.MatchesDate(date, tmpl.CreatedAt) {
			task := &taskdomain.Task{
				Title:            tmpl.Title,
				Description:      tmpl.Description,
				Status:           taskdomain.StatusNew,
				CreatedAt:        now,
				UpdatedAt:        now,
				PeriodicSourceID: &tmpl.ID,
			}
			toCreate = append(toCreate, task)
		}
	}

	if len(toCreate) == 0 {
		return []taskdomain.Task{}, nil
	}

	created, err := s.repo.CreateBatch(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if input.Periodicity != nil {
		if err := input.Periodicity.Valid(); err != nil {
			return CreateInput{}, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if input.Periodicity != nil {
		if err := input.Periodicity.Valid(); err != nil {
			return UpdateInput{}, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
	}

	return input, nil
}
