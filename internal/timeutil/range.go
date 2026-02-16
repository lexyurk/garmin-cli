package timeutil

import (
	"errors"
	"fmt"
	"time"
)

const DateLayout = "2006-01-02"

type RangeOptions struct {
	Date string
	From string
	To   string
	Days int
}

func ResolveDates(opts RangeOptions, now time.Time) ([]string, error) {
	// Enforce mutual exclusivity for clarity.
	used := 0
	if opts.Date != "" {
		used++
	}
	if opts.From != "" || opts.To != "" {
		used++
	}
	if opts.Days > 0 {
		used++
	}
	if used > 1 {
		return nil, errors.New("use only one of --date, --from/--to, or --days")
	}

	today := truncateToDate(now)

	switch {
	case opts.Date != "":
		d, err := parseDate(opts.Date)
		if err != nil {
			return nil, err
		}
		return []string{d.Format(DateLayout)}, nil

	case opts.Days > 0:
		if opts.Days <= 0 {
			return nil, errors.New("--days must be > 0")
		}
		start := today.AddDate(0, 0, -(opts.Days - 1))
		return formatDateRange(start, today), nil

	case opts.From != "" || opts.To != "":
		start := today
		end := today
		var err error
		if opts.From != "" {
			start, err = parseDate(opts.From)
			if err != nil {
				return nil, err
			}
		}
		if opts.To != "" {
			end, err = parseDate(opts.To)
			if err != nil {
				return nil, err
			}
		}
		if end.Before(start) {
			return nil, fmt.Errorf("--to (%s) is before --from (%s)", end.Format(DateLayout), start.Format(DateLayout))
		}
		return formatDateRange(start, end), nil

	default:
		return []string{today.Format(DateLayout)}, nil
	}
}

func formatDateRange(start, end time.Time) []string {
	start = truncateToDate(start)
	end = truncateToDate(end)

	out := make([]string, 0, int(end.Sub(start).Hours()/24)+1)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format(DateLayout))
	}
	return out
}

func parseDate(s string) (time.Time, error) {
	d, err := time.ParseInLocation(DateLayout, s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return truncateToDate(d), nil
}

func truncateToDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
