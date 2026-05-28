package workouts

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var repeatRe = regexp.MustCompile(`^\s*(\d+)\s*[xX]\s*\((.*)\)\s*$`)

// ParseSteps parses CLI step specifications into a step tree.
//
// Each spec is either an executable step ("<type> <duration>") or a repeat
// block ("Nx(<step>; <step>; ...)"). Examples:
//
//	"warmup 10min"
//	"interval 800m"
//	"recovery 90s"
//	"interval lap"
//	"4x(interval 800m; recovery 2min)"
//
// Durations: <n>min, <n>s/<n>sec (time); <n>m, <n>km (distance); "lap".
func ParseSteps(specs []string) ([]Step, error) {
	out := make([]Step, 0, len(specs))
	for _, spec := range specs {
		s, err := parseOneStep(spec)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func parseOneStep(spec string) (Step, error) {
	if m := repeatRe.FindStringSubmatch(spec); m != nil {
		iters, err := strconv.Atoi(m[1])
		if err != nil || iters < 1 {
			return Step{}, fmt.Errorf("invalid repeat count in %q", spec)
		}
		children := make([]Step, 0, 2)
		for _, part := range strings.Split(m[2], ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			cs, err := parseExecutable(part)
			if err != nil {
				return Step{}, err
			}
			children = append(children, cs)
		}
		if len(children) == 0 {
			return Step{}, fmt.Errorf("repeat %q contains no steps", spec)
		}
		return Step{Kind: RepeatStep, Iterations: iters, Children: children}, nil
	}
	return parseExecutable(spec)
}

func parseExecutable(spec string) (Step, error) {
	fields := strings.Fields(spec)
	if len(fields) != 2 {
		return Step{}, fmt.Errorf("step %q must be '<type> <duration>' (e.g. \"warmup 10min\")", spec)
	}
	typeKey := strings.ToLower(fields[0])
	if _, ok := stepTypeIDs[typeKey]; !ok || typeKey == "repeat" {
		return Step{}, fmt.Errorf("unknown step type %q (use warmup, cooldown, interval, recovery, rest, other)", fields[0])
	}

	st := Step{Kind: ExecutableStep, Type: typeKey}
	if strings.EqualFold(fields[1], "lap") {
		st.LapButton = true
		return st, nil
	}

	sec, meters, err := parseAmount(fields[1])
	if err != nil {
		return Step{}, err
	}
	st.DurationSec = sec
	st.DistanceM = meters
	return st, nil
}

func parseAmount(s string) (sec float64, meters float64, err error) {
	s = strings.ToLower(strings.TrimSpace(s))
	parse := func(suffix string) (float64, error) {
		return strconv.ParseFloat(strings.TrimSuffix(s, suffix), 64)
	}
	switch {
	case strings.HasSuffix(s, "min"):
		v, e := parse("min")
		if e != nil {
			break
		}
		return v * 60, 0, nil
	case strings.HasSuffix(s, "sec"):
		v, e := parse("sec")
		if e != nil {
			break
		}
		return v, 0, nil
	case strings.HasSuffix(s, "km"):
		v, e := parse("km")
		if e != nil {
			break
		}
		return 0, v * 1000, nil
	case strings.HasSuffix(s, "s"):
		v, e := parse("s")
		if e != nil {
			break
		}
		return v, 0, nil
	case strings.HasSuffix(s, "m"):
		v, e := parse("m")
		if e != nil {
			break
		}
		return 0, v, nil
	}
	return 0, 0, fmt.Errorf("invalid amount %q (use e.g. 10min, 90s, 800m, 5km, or lap)", s)
}
