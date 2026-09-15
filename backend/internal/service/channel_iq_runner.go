package service

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

const channelIQRunnerPollInterval = 15 * time.Second
const channelIQLeaderLockKey = "channel_iq_auto"
const channelIQLeaderLockTTL = 45 * time.Second

type ChannelIQRunner struct {
	svc       *ChannelIQService
	lockCache LeaderLockCache
	db        *sql.DB
	instance  string
	stopCh    chan struct{}
	stopOnce  sync.Once
}

func NewChannelIQRunner(svc *ChannelIQService) *ChannelIQRunner {
	return &ChannelIQRunner{svc: svc, stopCh: make(chan struct{}), instance: uuid.NewString()}
}

func ProvideChannelIQRunner(svc *ChannelIQService, lockCache LeaderLockCache, db *sql.DB) *ChannelIQRunner {
	r := NewChannelIQRunner(svc)
	r.lockCache = lockCache
	r.db = db
	r.Start()
	return r
}

func (r *ChannelIQRunner) Start() {
	go r.loop()
}

func (r *ChannelIQRunner) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
}

func (r *ChannelIQRunner) loop() {
	ticker := time.NewTicker(channelIQRunnerPollInterval)
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

func (r *ChannelIQRunner) tick() {
	if r.svc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	release, ok := tryAcquireSingletonLeaderLock(ctx, r.lockCache, r.db, channelIQLeaderLockKey, r.instance, channelIQLeaderLockTTL)
	if !ok {
		return
	}
	defer release()
	if err := r.svc.AutoTick(ctx); err != nil {
		log.Printf("channel iq auto tick: %v", err)
	}
}
