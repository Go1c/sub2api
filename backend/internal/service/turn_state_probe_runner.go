package service

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

const turnStateProbeRunnerPollInterval = 1 * time.Second
const turnStateProbeLeaderLockKey = "turn_state_probe"
const turnStateProbeLeaderLockTTL = 45 * time.Second
const turnStateProbeRunnerTickTimeout = 3 * time.Minute

type TurnStateProbeRunner struct {
	svc       *TurnStateProbeService
	lockCache LeaderLockCache
	db        *sql.DB
	instance  string
	stopCh    chan struct{}
	stopOnce  sync.Once
}

func NewTurnStateProbeRunner(svc *TurnStateProbeService) *TurnStateProbeRunner {
	return &TurnStateProbeRunner{svc: svc, stopCh: make(chan struct{}), instance: uuid.NewString()}
}

func ProvideTurnStateProbeRunner(svc *TurnStateProbeService, lockCache LeaderLockCache, db *sql.DB) *TurnStateProbeRunner {
	r := NewTurnStateProbeRunner(svc)
	r.lockCache = lockCache
	r.db = db
	r.Start()
	return r
}

func (r *TurnStateProbeRunner) Start() {
	go r.loop()
}

func (r *TurnStateProbeRunner) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
}

func (r *TurnStateProbeRunner) loop() {
	ticker := time.NewTicker(turnStateProbeRunnerPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.tick()
		}
	}
}

func (r *TurnStateProbeRunner) tick() {
	if r.svc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), turnStateProbeRunnerTickTimeout)
	defer cancel()
	release, ok := tryAcquireSingletonLeaderLock(ctx, r.lockCache, r.db, turnStateProbeLeaderLockKey, r.instance, turnStateProbeLeaderLockTTL)
	if !ok {
		return
	}
	defer release()
	if err := r.svc.HarvestDue(ctx); err != nil {
		log.Printf("turn state probe tick: %v", err)
	}
}
