package task

import (
	"fmt"
	"time"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID               int64        `json:"id"`
	Title            string       `json:"title"`
	Description      string       `json:"description"`
	Status           Status       `json:"status"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	Periodicity      *Periodicity `json:"periodicity,omitempty"`
	PeriodicSourceID *int64       `json:"periodic_source_id,omitempty"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

type PeriodicityType string

const (
	PeriodicityDaily         PeriodicityType = "daily"
	PeriodicityMonthly       PeriodicityType = "monthly"
	PeriodicitySpecificDates PeriodicityType = "specific_dates"
	PeriodicityEvenOdd       PeriodicityType = "even_odd"
)

type Periodicity struct {
	Type       PeriodicityType `json:"type"`
	Interval   int             `json:"interval,omitempty"`
	DayOfMonth int             `json:"day_of_month,omitempty"`
	Dates      []string        `json:"dates,omitempty"`
	EvenOdd    string          `json:"even_odd,omitempty"`
}

func (p *Periodicity) Valid() error {
	switch p.Type {
	case PeriodicityDaily:
		if p.Interval <= 0 {
			return fmt.Errorf("daily periodicity requires interval > 0")
		}
	case PeriodicityMonthly:
		if p.DayOfMonth < 1 || p.DayOfMonth > 31 {
			return fmt.Errorf("monthly periodicity requires day_of_month between 1 and 31")
		}
	case PeriodicitySpecificDates:
		if len(p.Dates) == 0 {
			return fmt.Errorf("specific_dates periodicity requires non-empty dates list")
		}
		for _, d := range p.Dates {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return fmt.Errorf("invalid date format %q, expected YYYY-MM-DD", d)
			}
		}
	case PeriodicityEvenOdd:
		if p.EvenOdd != "even" && p.EvenOdd != "odd" {
			return fmt.Errorf("even_odd periodicity requires even_odd to be \"even\" or \"odd\"")
		}
	default:
		return fmt.Errorf("unknown periodicity type %q", p.Type)
	}
	return nil
}

func (p *Periodicity) MatchesDate(date time.Time, createdAt time.Time) bool {
	switch p.Type {
	case PeriodicityDaily:
		days := int(date.Sub(createdAt.Truncate(24*time.Hour)).Hours() / 24)
		return days >= 0 && days%p.Interval == 0
	case PeriodicityMonthly:
		return date.Day() == p.DayOfMonth
	case PeriodicitySpecificDates:
		dateStr := date.Format("2006-01-02")
		for _, d := range p.Dates {
			if d == dateStr {
				return true
			}
		}
		return false
	case PeriodicityEvenOdd:
		day := date.Day()
		if p.EvenOdd == "even" {
			return day%2 == 0
		}
		return day%2 != 0
	default:
		return false
	}
}
