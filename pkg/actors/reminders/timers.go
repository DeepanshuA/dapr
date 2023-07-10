/*
Copyright 2023 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reminders

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/utils/clock"

	actorsCore "github.com/dapr/dapr/pkg/actors/core"
	actorsCoreReminder "github.com/dapr/dapr/pkg/actors/core/reminder"
	diag "github.com/dapr/dapr/pkg/diagnostics"
)

type ActorsTimers struct {
	activeTimersLock      sync.RWMutex
	clock                 *clock.WithTicker
	actorsTable           *sync.Map
	activeTimers          *sync.Map
	actorsReminders       actorsCore.Reminders
	activeTimersCountLock sync.RWMutex
	activeTimersCount     map[string]*int64
}

type TimerOpts struct {
	Clock             *clock.WithTicker
	ActorsTable       *sync.Map
	ActiveTimers      *sync.Map
	ActorsReminders   actorsCore.Reminders
	ActiveTimersCount map[string]*int64
}

func NewTimers(opts TimerOpts) actorsCore.Timers {
	return &ActorsTimers{
		clock:             opts.Clock,
		actorsTable:       opts.ActorsTable,
		activeTimers:      opts.ActiveTimers,
		actorsReminders:   opts.ActorsReminders,
		activeTimersCount: opts.ActiveTimersCount,
	}
}

func (a *ActorsTimers) CreateTimer(ctx context.Context, req *CreateTimerRequest) error {
	reminder, err := NewReminderFromCreateTimerRequest(req, (*a.clock).Now())
	if err != nil {
		return err
	}

	a.activeTimersLock.Lock()
	defer a.activeTimersLock.Unlock()

	actorKey := reminder.ActorKey()
	timerKey := reminder.Key()

	_, exists := a.actorsTable.Load(actorKey)
	if !exists {
		return fmt.Errorf("can't create timer for actor %s: actor not activated", actorKey)
	}

	stopChan, exists := a.activeTimers.Load(timerKey)
	if exists {
		close(stopChan.(chan struct{}))
	}

	log.Debugf("Create timer '%s' dueTime:'%s' period:'%s' ttl:'%v'",
		timerKey, reminder.DueTime, reminder.Period, reminder.ExpirationTime)

	stop := make(chan struct{}, 1)
	a.activeTimers.Store(timerKey, stop)
	a.updateActiveTimersCount(req.ActorType, 1)

	go func() {
		var (
			ttlTimer, nextTimer clock.Timer
			ttlTimerC           <-chan time.Time
			err                 error
		)

		if !reminder.ExpirationTime.IsZero() {
			ttlTimer = (*a.clock).NewTimer(reminder.ExpirationTime.Sub((*a.clock).Now()))
			ttlTimerC = ttlTimer.C()
		}

		nextTimer = (*a.clock).NewTimer(reminder.NextTick().Sub((*a.clock).Now()))
		defer func() {
			if nextTimer != nil && !nextTimer.Stop() {
				<-nextTimer.C()
			}
			if ttlTimer != nil && !ttlTimer.Stop() {
				<-ttlTimer.C()
			}
		}()

	L:
		for {
			select {
			case <-nextTimer.C():
				// noop
			case <-ttlTimerC:
				// timer has expired; proceed with deletion
				log.Infof("Timer %s with parameters: dueTime: %s, period: %s, TTL: %s has expired", timerKey, req.DueTime, req.Period, req.TTL)
				ttlTimer = nil
				break L
			case <-stop:
				// timer has been already deleted
				log.Infof("Timer %s with parameters: dueTime: %s, period: %s, TTL: %s has been deleted", timerKey, req.DueTime, req.Period, req.TTL)
				return
			}

			if _, exists := a.actorsTable.Load(actorKey); exists {
				err = a.actorsReminders.ExecuteReminder(reminder, true)
				diag.DefaultMonitoring.ActorTimerFired(req.ActorType, err == nil)
				if err != nil {
					log.Errorf("error invoking timer on actor %s: %s", actorKey, err)
				}
			} else {
				log.Errorf("Could not find active timer %s", timerKey)
				nextTimer = nil
				return
			}

			if reminder.TickExecuted() {
				log.Infof("Timer %s has been completed", timerKey)
				nextTimer = nil
				break L
			}

			nextTimer.Reset(reminder.NextTick().Sub((*a.clock).Now()))
		}

		err = a.DeleteTimer(ctx, &actorsCoreReminder.DeleteTimerRequest{
			Name:      req.Name,
			ActorID:   req.ActorID,
			ActorType: req.ActorType,
		})
		if err != nil {
			log.Errorf("error deleting timer %s: %v", timerKey, err)
		}
	}()
	return nil
}

func (a *ActorsTimers) DeleteTimer(ctx context.Context, req *actorsCoreReminder.DeleteTimerRequest) error {
	actorKey := constructCompositeKey(req.ActorType, req.ActorID)
	timerKey := constructCompositeKey(actorKey, req.Name)

	stopChan, exists := a.activeTimers.Load(timerKey)
	if exists {
		close(stopChan.(chan struct{}))
		a.activeTimers.Delete(timerKey)
		a.updateActiveTimersCount(req.ActorType, -1)
	}

	return nil
}

func (a *ActorsTimers) updateActiveTimersCount(actorType string, inc int64) {
	a.activeTimersCountLock.RLock()
	_, ok := a.activeTimersCount[actorType]
	a.activeTimersCountLock.RUnlock()
	if !ok {
		a.activeTimersCountLock.Lock()
		if _, ok = a.activeTimersCount[actorType]; !ok { // re-check
			a.activeTimersCount[actorType] = new(int64)
		}
		a.activeTimersCountLock.Unlock()
	}

	diag.DefaultMonitoring.ActorTimers(actorType, atomic.AddInt64(a.activeTimersCount[actorType], inc))
}
