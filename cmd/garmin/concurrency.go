package main

import (
	"context"
	"sync"
)

func mapDatesConcurrently[T any](
	ctx context.Context,
	dates []string,
	limit int,
	fn func(ctx context.Context, date string) (T, error),
) ([]T, error) {
	if limit <= 0 {
		limit = 4
	}
	if len(dates) == 0 {
		return []T{}, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		idx int
		val T
		err error
	}

	sem := make(chan struct{}, limit)
	resCh := make(chan result, len(dates))
	var wg sync.WaitGroup

	for i, d := range dates {
		wg.Add(1)
		go func(idx int, date string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				var zero T
				resCh <- result{idx: idx, val: zero, err: ctx.Err()}
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				var zero T
				resCh <- result{idx: idx, val: zero, err: ctx.Err()}
				return
			default:
			}

			v, err := fn(ctx, date)
			if err != nil {
				cancel()
			}
			resCh <- result{idx: idx, val: v, err: err}
		}(i, d)
	}

	wg.Wait()
	close(resCh)

	out := make([]T, len(dates))
	var firstErr error
	for r := range resCh {
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		out[r.idx] = r.val
	}

	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

