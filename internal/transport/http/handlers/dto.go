package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type periodicityDTO struct {
	Type       taskdomain.PeriodicityType `json:"type"`
	Interval   int                        `json:"interval,omitempty"`
	DayOfMonth int                        `json:"day_of_month,omitempty"`
	Dates      []string                   `json:"dates,omitempty"`
	EvenOdd    string                     `json:"even_odd,omitempty"`
}

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	Periodicity *periodicityDTO   `json:"periodicity,omitempty"`
}

type taskDTO struct {
	ID               int64             `json:"id"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Status           taskdomain.Status `json:"status"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	Periodicity      *periodicityDTO   `json:"periodicity,omitempty"`
	PeriodicSourceID *int64            `json:"periodic_source_id,omitempty"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:               task.ID,
		Title:            task.Title,
		Description:      task.Description,
		Status:           task.Status,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
		PeriodicSourceID: task.PeriodicSourceID,
	}

	if task.Periodicity != nil {
		dto.Periodicity = &periodicityDTO{
			Type:       task.Periodicity.Type,
			Interval:   task.Periodicity.Interval,
			DayOfMonth: task.Periodicity.DayOfMonth,
			Dates:      task.Periodicity.Dates,
			EvenOdd:    task.Periodicity.EvenOdd,
		}
	}

	return dto
}

func periodicityDTOToDomain(dto *periodicityDTO) *taskdomain.Periodicity {
	if dto == nil {
		return nil
	}
	return &taskdomain.Periodicity{
		Type:       dto.Type,
		Interval:   dto.Interval,
		DayOfMonth: dto.DayOfMonth,
		Dates:      dto.Dates,
		EvenOdd:    dto.EvenOdd,
	}
}
