package jobs

import "errors"

// getOrCreateWorkerPool gets an existing worker pool or creates a new one
func (svc *JobService) getOrCreateWorkerPool(queueName string) (*workerPool, error) {
	svc.workersMu.RLock()
	pool, exists := svc.workers[queueName]
	svc.workersMu.RUnlock()

	if exists {
		return pool, nil
	}

	return svc.ensureWorkerPool(queueName)
}

// ensureWorkerPool ensures a worker pool exists for the given queue
func (svc *JobService) ensureWorkerPool(queueName string) (*workerPool, error) {
	svc.workersMu.Lock()
	defer svc.workersMu.Unlock()

	// Check again (double-check pattern for concurrency safety)
	if pool, exists := svc.workers[queueName]; exists {
		return pool, nil
	}

	// Get worker count from config (with default)
	workerCount := svc.config.Jobs.DefaultWorkers
	if queueConfig, ok := svc.config.Jobs.Queues[queueName]; ok {
		workerCount = queueConfig.Workers
	}

	if workerCount <= 0 {
		return nil, errors.Join(ErrInternal, errors.New("invalid worker count"))
	}

	pool := newWorkerPool(queueName, svc, workerCount)
	svc.workers[queueName] = pool

	return pool, nil
}
