package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	jobChannelsMu sync.RWMutex
	jobChannels   = map[string]chan Job{
		JobTypeEmailSend:          make(chan Job, 5),
		JobTypeSecurityEventLog:   make(chan Job, 20),
		JobTypeWebhookDispatch:    make(chan Job, 10),
		JobTypeNotificationCreate: make(chan Job, 20),
	}
)

type WorkerManager struct {
	workerID  string
	repo      Repository
	db        *sql.DB
	executors map[string]JobExecutor
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	shutdown  chan struct{}
	once      sync.Once
}

func NewWorkerManager(repo Repository, db *sql.DB, executors map[string]JobExecutor) *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerManager{
		workerID:  "worker-" + uuid.New().String(),
		repo:      repo,
		db:        db,
		executors: executors,
		ctx:       ctx,
		cancel:    cancel,
		shutdown:  make(chan struct{}),
	}
}

func (wm *WorkerManager) Start() {
	slog.Info("Starting queue WorkerManager", "worker_id", wm.workerID)

	jobChannelsMu.RLock()
	for jt, ch := range jobChannels {
		limit := cap(ch)
		for i := 0; i < limit; i++ {
			wm.wg.Add(1)
			go func(c chan Job) {
				defer wm.wg.Done()
				for {
					select {
					case <-wm.ctx.Done():
						return
					case job, ok := <-c:
						if !ok {
							return
						}
						wm.executeJob(job)
					}
				}
			}(ch)
		}
		slog.Info("Started workers for job type", "count", limit, "job_type", jt)
	}
	jobChannelsMu.RUnlock()

	wm.wg.Add(1)
	go func() {
		defer wm.wg.Done()
		ticker := time.NewTicker(1500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-wm.ctx.Done():
				return
			case <-ticker.C:
				wm.pollAndProcess()
			}
		}
	}()

	wm.wg.Add(1)
	go func() {
		defer wm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-wm.ctx.Done():
				return
			case <-ticker.C:
				affected, err := wm.repo.RecoverStuck(wm.ctx)
				if err != nil {
					slog.Error("Error recovering stuck jobs", "error", err)
				} else if affected > 0 {
					slog.Info("Recovered stuck jobs back to pending state", "count", affected)
				}
			}
		}
	}()

	wm.wg.Add(1)
	go func() {
		defer wm.wg.Done()
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-wm.ctx.Done():
				return
			case <-ticker.C:
				pruned, err := wm.repo.PruneExpired(wm.ctx, 24*time.Hour)
				if err != nil {
					slog.Error("Error pruning expired jobs", "error", err)
					continue
				} else if pruned > 0 {
					slog.Info("Pruned expired jobs older than 24 hours", "count", pruned)
				}
				if err := wm.repo.Vacuum(wm.ctx); err != nil {
					slog.Error("Error analyzing jobs table", "error", err)
				}
			}
		}
	}()
}

func (wm *WorkerManager) Stop() {
	wm.once.Do(func() {
		slog.Info("Shutdown triggered. Stopping queue worker manager...")
		wm.cancel()

		done := make(chan struct{})
		go func() {
			wm.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("All worker routines completed cleanly")
		case <-time.After(30 * time.Second):
			slog.Warn("Worker shutdown timed out — some jobs may remain in processing state")
		}
		close(wm.shutdown)
	})
}

func (wm *WorkerManager) pollAndProcess() {
	var totalAvailable int

	jobChannelsMu.RLock()
	for _, ch := range jobChannels {
		available := cap(ch) - len(ch)
		if available > 0 {
			totalAvailable += available
		}
	}

	if totalAvailable <= 0 {
		jobChannelsMu.RUnlock()
		return
	}

	batchSize := totalAvailable
	jobChannelsMu.RUnlock()
	if batchSize > 20 {
		batchSize = 20
	}

	jobs, err := wm.repo.Claim(wm.ctx, wm.workerID, batchSize)
	if err != nil {
		slog.Error("Error claiming jobs", "error", err)
		return
	}

	for i, job := range jobs {
		jobType := job.Type

		jobChannelsMu.RLock()
		ch, exists := jobChannels[jobType]
		jobChannelsMu.RUnlock()
		if !exists {
			slog.Warn("Unregistered queue channel/job type. Completing immediately to prevent orphaning.", "job_type", jobType)
			dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = wm.repo.Complete(dbCtx, job.ID)
			dbCancel()
			continue
		}

		select {
		case <-wm.ctx.Done():
			now := time.Now()
			for j := i; j < len(jobs); j++ {
				dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = wm.repo.Fail(dbCtx, jobs[j].ID, "worker shutdown during dispatch", &now)
				dbCancel()
			}
			return
		case ch <- job:
		default:
			slog.Warn("Channel full, dropping load for job. Will be recovered by RecoverStuck.", "job_type", jobType, "job_id", job.ID)
		}
	}
}

func (wm *WorkerManager) executeJob(j Job) {
	start := time.Now()

	if j.Status == "done" || j.Status == "failed" {
		slog.Info("Job already completed. Skipping.", "job_id", j.ID)
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dbCancel()
		_ = wm.repo.Complete(dbCtx, j.ID)
		return
	}

	executor, registered := wm.executors[j.Type]

	if !registered {
		slog.Error("Unregistered executor for job type", "job_type", j.Type)
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dbCancel()
		_ = wm.repo.Fail(dbCtx, j.ID, "unregistered job executor", nil)
		return
	}

	execCtx, execCancel := context.WithCancel(wm.ctx)
	defer execCancel()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-ticker.C:
				err := wm.repo.Heartbeat(execCtx, j.ID, wm.workerID)
				if err != nil {
					slog.Error("Heartbeat error", "job_id", j.ID, "error", err)
				}
			}
		}
	}()

	var execErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				execErr = fmt.Errorf("job panicked: %v", r)
			}
		}()
		execErr = executor.Execute(execCtx, j.Payload)
	}()

	execCancel()
	duration := time.Since(start)

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	if execErr != nil {
		slog.Error("Job failed", "job_id", j.ID, "job_type", j.Type, "error", execErr)

		backoffSec := int(math.Min(math.Pow(2, float64(j.RetryCount)), 60))
		nextRetry := time.Now().Add(time.Duration(backoffSec) * time.Second)

		if j.RetryCount < j.MaxRetries {
			_ = wm.repo.Fail(dbCtx, j.ID, execErr.Error(), &nextRetry)
			slog.Warn("Job retry", "event_type", "JOB_RETRY", "latency_ms", duration.Milliseconds())
		} else {
			dlqPayload, _ := json.Marshal(map[string]interface{}{
				"job_id":   j.ID,
				"job_type": j.Type,
				"payload":  string(j.Payload),
				"error":    execErr.Error(),
				"retries":  j.RetryCount,
			})
			_, _ = wm.db.ExecContext(dbCtx,
				`INSERT INTO dead_letter_jobs (original_job_id, type, payload, error_message)
				 VALUES ($1, $2, $3, $4)`,
				j.ID, j.Type, dlqPayload, execErr.Error(),
			)
			_ = wm.repo.Fail(dbCtx, j.ID, execErr.Error(), nil)
			slog.Error("Job failed", "event_type", "JOB_FAILED", "latency_ms", duration.Milliseconds())
		}
	} else {
		_ = wm.repo.Complete(dbCtx, j.ID)
		slog.Info("Job succeeded", "job_id", j.ID, "job_type", j.Type, "latency_ms", duration.Milliseconds())
	}
}

func StartWorker(repo Repository, db *sql.DB, executors map[string]JobExecutor) *WorkerManager {
	m := NewWorkerManager(repo, db, executors)
	m.Start()
	return m
}
