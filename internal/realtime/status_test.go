package realtime

import (
	"sort"
	"testing"
	"time"
)

func TestServiceAggregatesTerminalStatusesAndClearsTaskOverride(t *testing.T) {
	service := New(Options{})
	service.RegisterTerminal("task-1", "terminal-working")
	service.RegisterTerminal("task-1", "terminal-unread")
	service.SetTaskStatus("task-1", StatusError)
	if got := service.TaskStatus("task-1"); got != StatusError {
		t.Fatalf("直接任务状态 = %q，期望 %q", got, StatusError)
	}

	service.SetTerminalStatus("task-1", "terminal-working", StatusWorking)
	if got := service.TaskStatus("task-1"); got != StatusWorking {
		t.Fatalf("工作中终端覆盖后的任务状态 = %q，期望 %q", got, StatusWorking)
	}
	service.SetTerminalStatus("task-1", "terminal-unread", StatusUnread)
	if got := service.TaskStatus("task-1"); got != StatusUnread {
		t.Fatalf("未读优先级任务状态 = %q，期望 %q", got, StatusUnread)
	}
	service.SetTerminalStatus("task-1", "terminal-working", StatusError)
	if got := service.TaskStatus("task-1"); got != StatusError {
		t.Fatalf("异常优先级任务状态 = %q，期望 %q", got, StatusError)
	}
}

func TestServiceMarksTitleActivityUnreadAfterSilenceAndClearsOnSelection(t *testing.T) {
	clock := &fakeClock{}
	service := New(Options{Clock: clock})
	service.RegisterTerminal("task-1", "terminal-1")

	service.ReportTitleActivity("task-1", "terminal-1")
	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusWorking {
		t.Fatalf("标题活动后终端状态 = %q，期望 %q", got, StatusWorking)
	}

	clock.Advance(1500 * time.Millisecond)
	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusUnread {
		t.Fatalf("未选中终端静默后的状态 = %q，期望 %q", got, StatusUnread)
	}

	service.SelectTerminal("task-1", "terminal-1")
	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusIdle {
		t.Fatalf("选择未读终端后的状态 = %q，期望 %q", got, StatusIdle)
	}
}

func TestServiceMarksSelectedTerminalIdleAfterTitleSilence(t *testing.T) {
	clock := &fakeClock{}
	service := New(Options{Clock: clock})
	service.RegisterTerminal("task-1", "terminal-1")
	service.SelectTerminal("task-1", "terminal-1")

	service.ReportTitleActivity("task-1", "terminal-1")
	clock.Advance(1500 * time.Millisecond)

	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusIdle {
		t.Fatalf("选中终端静默后的状态 = %q，期望 %q", got, StatusIdle)
	}
}

func TestServiceIgnoresStaleTitleActivityTimers(t *testing.T) {
	clock := &fakeClock{}
	service := New(Options{Clock: clock})
	service.RegisterTerminal("task-1", "terminal-1")

	service.ReportTitleActivity("task-1", "terminal-1")
	clock.Advance(time.Second)
	service.ReportTitleActivity("task-1", "terminal-1")
	clock.Advance(500 * time.Millisecond)

	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusWorking {
		t.Fatalf("旧计时器到期后的状态 = %q，期望 %q", got, StatusWorking)
	}
	clock.Advance(time.Second)
	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusUnread {
		t.Fatalf("新计时器到期后的状态 = %q，期望 %q", got, StatusUnread)
	}
}

func TestServiceResetsStatusesWhenModeChanges(t *testing.T) {
	service := New(Options{})
	service.RegisterTerminal("task-1", "terminal-1")
	service.SetTerminalStatus("task-1", "terminal-1", StatusWorking)
	service.SetTaskStatus("task-1", StatusError)

	service.SetMode(ModeHTTP)

	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusIdle {
		t.Errorf("切换模式后的终端状态 = %q，期望 %q", got, StatusIdle)
	}
	if got := service.TaskStatus("task-1"); got != StatusIdle {
		t.Errorf("切换模式后的任务状态 = %q，期望 %q", got, StatusIdle)
	}
}

func TestServicePublishesIncreasingVersions(t *testing.T) {
	var events []Event
	service := New(Options{Publish: func(event Event) {
		events = append(events, event)
	}})
	service.RegisterTerminal("task-1", "terminal-1")
	service.SetTerminalStatus("task-1", "terminal-1", StatusWorking)

	if len(events) < 2 {
		t.Fatalf("状态事件数量 = %d，期望至少 2", len(events))
	}
	if events[len(events)-1].Version <= events[0].Version {
		t.Errorf("状态事件版本未递增：首个 = %d，最后 = %d", events[0].Version, events[len(events)-1].Version)
	}
}

type fakeClock struct {
	now    time.Duration
	timers []*fakeTimer
}

func (clock *fakeClock) AfterFunc(delay time.Duration, callback func()) Timer {
	timer := &fakeTimer{at: clock.now + delay, callback: callback}
	clock.timers = append(clock.timers, timer)
	return timer
}

func (clock *fakeClock) Advance(duration time.Duration) {
	clock.now += duration
	for {
		sort.SliceStable(clock.timers, func(left, right int) bool {
			return clock.timers[left].at < clock.timers[right].at
		})
		var next *fakeTimer
		for _, timer := range clock.timers {
			if !timer.stopped && !timer.fired && timer.at <= clock.now {
				next = timer
				break
			}
		}
		if next == nil {
			return
		}
		next.fired = true
		next.callback()
	}
}

type fakeTimer struct {
	at       time.Duration
	callback func()
	stopped  bool
	fired    bool
}

func (timer *fakeTimer) Stop() bool {
	if timer.stopped || timer.fired {
		return false
	}
	timer.stopped = true
	return true
}
