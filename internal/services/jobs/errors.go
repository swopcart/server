package jobs

import "errors"

var (
	ErrInternal = errors.New("internal error")

	ErrHandlerNotFound          = errors.New("job handler not found")
	ErrHandlerAlreadyRegistered = errors.New("job handler already registered")
	ErrInvalidSchedule          = errors.New("invalid cron schedule")
	ErrJobNotFound              = errors.New("job not found")
	ErrExecutionNotFound        = errors.New("job execution not found")
	ErrQueueFull                = errors.New("job queue is full")
)
