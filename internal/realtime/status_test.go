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

func TestServiceMarksOutputActivityUnreadAfterSilence(t *testing.T) {
	clock := &fakeClock{}
	service := New(Options{Clock: clock, Mode: ModeOutputChange})
	service.RegisterTerminal("task-1", "terminal-1")

	if !service.ReportOutputActivity("task-1", "terminal-1") {
		t.Fatal("ReportOutputActivity() = false，期望接受输出活动")
	}
	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusWorking {
		t.Fatalf("输出活动后终端状态 = %q，期望 %q", got, StatusWorking)
	}

	clock.Advance(TitleActivityTimeout)
	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusUnread {
		t.Fatalf("输出静默后的终端状态 = %q，期望 %q", got, StatusUnread)
	}
}

func TestServiceKeepsOutputActivityWorkingUntilLatestOutputIsSilent(t *testing.T) {
	clock := &fakeClock{}
	service := New(Options{Clock: clock, Mode: ModeOutputChange})
	service.RegisterTerminal("task-1", "terminal-1")

	service.ReportOutputActivity("task-1", "terminal-1")
	clock.Advance(time.Second)
	service.ReportOutputActivity("task-1", "terminal-1")
	if got := clock.afterFuncCalls; got != 1 {
		t.Fatalf("连续输出前创建计时器次数 = %d，期望 1", got)
	}

	clock.Advance(500 * time.Millisecond)
	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusWorking {
		t.Fatalf("首次计时器到期后的终端状态 = %q，期望 %q", got, StatusWorking)
	}
	clock.Advance(time.Second)
	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusUnread {
		t.Fatalf("最新输出静默后的终端状态 = %q，期望 %q", got, StatusUnread)
	}
}

func TestServiceMarksSelectedOutputTerminalIdleAfterSilence(t *testing.T) {
	clock := &fakeClock{}
	service := New(Options{Clock: clock, Mode: ModeOutputChange})
	service.RegisterTerminal("task-1", "terminal-1")
	service.SelectTerminal("task-1", "terminal-1")

	service.ReportOutputActivity("task-1", "terminal-1")
	clock.Advance(TitleActivityTimeout)

	if got := service.TerminalStatus("task-1", "terminal-1"); got != StatusIdle {
		t.Fatalf("选中终端输出静默后的状态 = %q，期望 %q", got, StatusIdle)
	}
}

func TestServiceAcceptsOnlyMatchingAutomaticActivitySource(t *testing.T) {
	titleService := New(Options{Mode: ModeTitleChange})
	titleService.RegisterTerminal("task-1", "terminal-1")
	if titleService.ReportOutputActivity("task-1", "terminal-1") {
		t.Fatal("标题方式错误接受输出活动")
	}

	outputService := New(Options{Mode: ModeOutputChange})
	outputService.RegisterTerminal("task-1", "terminal-1")
	if outputService.ReportTitleActivity("task-1", "terminal-1") {
		t.Fatal("输出方式错误接受标题活动")
	}

	httpService := New(Options{Mode: ModeHTTP})
	httpService.RegisterTerminal("task-1", "terminal-1")
	if httpService.ReportOutputActivity("task-1", "terminal-1") {
		t.Fatal("HTTP 方式错误接受输出活动")
	}
}

func TestServiceDoesNotRepublishWorkingOutputStatus(t *testing.T) {
	clock := &fakeClock{}
	events := make([]Event, 0)
	service := New(Options{Clock: clock, Mode: ModeOutputChange, Publish: func(event Event) {
		events = append(events, event)
	}})
	service.RegisterTerminal("task-1", "terminal-1")

	service.ReportOutputActivity("task-1", "terminal-1")
	service.ReportOutputActivity("task-1", "terminal-1")

	if got := len(events); got != 2 {
		t.Fatalf("连续输出后的状态事件数量 = %d，期望 2", got)
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
	now            time.Duration
	timers         []*fakeTimer
	afterFuncCalls int
}

func (clock *fakeClock) AfterFunc(delay time.Duration, callback func()) Timer {
	clock.afterFuncCalls++
	timer := &fakeTimer{at: clock.now + delay, callback: callback}
	clock.timers = append(clock.timers, timer)
	return timer
}

func (clock *fakeClock) Now() time.Time {
	return time.Unix(0, 0).Add(clock.now)
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
