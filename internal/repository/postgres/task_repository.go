package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	periodicityJSON, err := marshalPeriodicity(task.Periodicity)
	if err != nil {
		return nil, err
	}

	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at, periodicity, periodic_source_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, description, status, created_at, updated_at, periodicity, periodic_source_id
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt, periodicityJSON, task.PeriodicSourceID)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at, periodicity, periodic_source_id
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	periodicityJSON, err := marshalPeriodicity(task.Periodicity)
	if err != nil {
		return nil, err
	}

	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			updated_at = $4,
			periodicity = $5
		WHERE id = $6
		RETURNING id, title, description, status, created_at, updated_at, periodicity, periodic_source_id
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.UpdatedAt, periodicityJSON, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at, periodicity, periodic_source_id
		FROM tasks
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) ListPeriodicTemplates(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at, periodicity, periodic_source_id
		FROM tasks
		WHERE periodicity IS NOT NULL
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) CreateBatch(ctx context.Context, tasks []*taskdomain.Task) ([]taskdomain.Task, error) {
	if len(tasks) == 0 {
		return []taskdomain.Task{}, nil
	}

	valueStrings := make([]string, 0, len(tasks))
	args := make([]any, 0, len(tasks)*7)

	for i, t := range tasks {
		periodicityJSON, err := marshalPeriodicity(t.Periodicity)
		if err != nil {
			return nil, err
		}

		base := i * 7
		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7,
		))
		args = append(args, t.Title, t.Description, t.Status, t.CreatedAt, t.UpdatedAt, periodicityJSON, t.PeriodicSourceID)
	}

	query := `
		INSERT INTO tasks (title, description, status, created_at, updated_at, periodicity, periodic_source_id)
		VALUES ` + strings.Join(valueStrings, ", ") + `
		RETURNING id, title, description, status, created_at, updated_at, periodicity, periodic_source_id
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	created := make([]taskdomain.Task, 0, len(tasks))
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		created = append(created, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return created, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task            taskdomain.Task
		status          string
		periodicityJSON []byte
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&periodicityJSON,
		&task.PeriodicSourceID,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	if len(periodicityJSON) > 0 {
		var p taskdomain.Periodicity
		if err := json.Unmarshal(periodicityJSON, &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal periodicity: %w", err)
		}
		task.Periodicity = &p
	}

	return &task, nil
}

func marshalPeriodicity(p *taskdomain.Periodicity) ([]byte, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}
