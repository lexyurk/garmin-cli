package main

type cliError struct {
	err      error
	rendered bool
}

func (e *cliError) Error() string {
	return e.err.Error()
}

func (e *cliError) Unwrap() error {
	return e.err
}

func renderedError(err error) error {
	if err == nil {
		return nil
	}
	return &cliError{err: err, rendered: true}
}
