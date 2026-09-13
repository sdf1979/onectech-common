package onectechcommon

import (
	"time"

	agg "github.com/sdf1979/onectech-common/aggregator"
)

type dblockState struct {
	excp *EventLog
	db   *EventLog
	call *EventLog
}

type Analysis struct {
	Ttimeout   *Node
	Tdeadlock  *Node
	Dblock     *Node
	Call       *Node
	DbQuery    *Node
	Excp       *Node
	normalizer *QueryNormalizer

	timeStart time.Time
	timeEnd   time.Time

	ttimeoutUsrEvent map[ThreadSessionKey]*EventLog
	callUsrEvent     map[ThreadSessionKey]*EventLog
	excpUsrEvent     map[ThreadSessionKey]*EventLog
	dblockUserEvent  map[ThreadSessionKey]*dblockState
}

func NewAnalysis() *Analysis {
	return &Analysis{
		Ttimeout:  NewNode(&agg.Counter{}),
		Tdeadlock: NewNode(&agg.Counter{}),
		Dblock:    NewNode(&agg.Counter{}),
		Call:      NewNode(&agg.CallCounter{}),
		DbQuery:   NewNode(&agg.DurationCounter{}),
		Excp:      NewNode(&agg.Counter{}),

		normalizer: NewQueryNormalizer(),

		ttimeoutUsrEvent: make(map[ThreadSessionKey]*EventLog),
		callUsrEvent:     make(map[ThreadSessionKey]*EventLog),
		excpUsrEvent:     make(map[ThreadSessionKey]*EventLog),
		dblockUserEvent:  make(map[ThreadSessionKey]*dblockState),
	}
}

func (a *Analysis) Process(out <-chan *EventLog) {
	for event := range out {
		key := event.ThreadSessionKey()
		a.updateTime(event)
		a.processEvent(key, event)
	}
	a.finalize()
}

func (a *Analysis) TimeStart() time.Time {
	return a.timeStart
}

func (a *Analysis) TimeEnd() time.Time {
	return a.timeEnd
}

func (a *Analysis) updateTime(event *EventLog) {
	t := event.Time()
	if a.timeStart.IsZero() || a.timeStart.After(t) {
		a.timeStart = t
	}
	if a.timeEnd.IsZero() || a.timeEnd.Before(t) {
		a.timeEnd = t
	}
}

func (a *Analysis) processEvent(key ThreadSessionKey, event *EventLog) {
	switch event.Name() {
	case EXCP:
		a.handleException(key, event)
	case CALL:
		a.handleCall(key, event)
	case CONTEXT:
		a.handleContext(key, event)
	case TDEADLOCK:
		a.handleTdeadlock(event)
	case TTIMEOUT:
		a.ttimeoutUsrEvent[key] = event
	case TLOCK:
		a.handleTlock(key, event)
	case DBMSSQL, DBPOSTGRS:
		a.handleDbQuery(key, event)
	}
}

func (a *Analysis) handleException(key ThreadSessionKey, event *EventLog) {
	exception := event.Value(EXCEPTION)
	descr := event.Descr()

	if ContainsAny(descr,
		"Выполнение оператора KILL не привело к ошибке СУБД",
		"Сеанс работы завершен администратором",
		"Сеанс отсутствует или удален",
	) {
		return
	}

	ib := event.Value(P_PROCESS_NAME)
	user := event.Value(USR)
	context := event.Context()

	if context != "" {
		a.insertException(ib, descr, context)
	} else if user != "" {
		a.excpUsrEvent[key] = event
	}

	if exception == DATABASE_EXCEPTION &&
		ContainsAny(descr, DB_LOCK_MSSQL, DB_LOCK_POSTGRS) {
		a.dblockUserEvent[key] = &dblockState{
			excp: event,
			db:   &EventLog{},
			call: &EventLog{},
		}
	}
}

func (a *Analysis) handleCall(key ThreadSessionKey, event *EventLog) {
	user := event.Value(USR)
	callContext := event.CallContext()

	if callContext != "" {
		ib := event.Value(P_PROCESS_NAME)
		a.insertCall(ib, callContext, event)
	} else if user != "" {
		a.callUsrEvent[key] = event
	}

	state, ok := a.dblockUserEvent[key]
	if !ok {
		return
	}

	ib := event.Value(P_PROCESS_NAME)
	context := event.Context()
	retExcp := event.Value(RET_EXCP)

	if ContainsAny(retExcp, DB_LOCK_MSSQL, DB_LOCK_POSTGRS) {
		if context != "" {
			sql := state.db.Value(SQL)
			digest := a.normalizer.Digest(sql, state.db.Name())
			a.insertDblock(ib, retExcp, digest, context)
			delete(a.dblockUserEvent, key)
		} else {
			state.call = event
		}
		return
	}

	if context == "" {
		context = state.excp.Context()
	}
	if context == "" {
		context = state.db.Context()
	}

	retExcp = state.excp.Descr()
	a.insertDblock(ib, retExcp, "Request text not found", context)
	delete(a.dblockUserEvent, key)
}

func (a *Analysis) handleContext(key ThreadSessionKey, event *EventLog) {
	a.handleCallContext(key, event)
	a.handleExceptionContext(key, event)
	a.handleTtimeoutContext(key, event)
	a.handleDblockContext(key, event)
}

func (a *Analysis) handleCallContext(key ThreadSessionKey, event *EventLog) {
	eventCall, ok := a.callUsrEvent[key]
	if !ok {
		return
	}

	context := event.Context()
	if context != "" {
		ib := eventCall.Value(P_PROCESS_NAME)
		a.insertCall(ib, context, eventCall)
	}
	delete(a.callUsrEvent, key)
}

func (a *Analysis) handleExceptionContext(key ThreadSessionKey, event *EventLog) {
	eventExcp, ok := a.excpUsrEvent[key]
	if !ok {
		return
	}

	context := event.Context()
	if context != "" {
		ib := eventExcp.Value(P_PROCESS_NAME)
		descr := eventExcp.Descr()
		a.insertException(ib, descr, context)
	}
	delete(a.excpUsrEvent, key)
}

func (a *Analysis) handleTtimeoutContext(key ThreadSessionKey, event *EventLog) {
	eventLock, ok := a.ttimeoutUsrEvent[key]
	if !ok {
		return
	}

	ib := eventLock.Value(P_PROCESS_NAME)
	regions := eventLock.Value(REGIONS)
	context := event.Context()

	a.Ttimeout.Insert([]string{ib, regions, context}, agg.UpdateContext{Count: 1})
	delete(a.ttimeoutUsrEvent, key)
}

func (a *Analysis) handleDblockContext(key ThreadSessionKey, event *EventLog) {
	state, ok := a.dblockUserEvent[key]
	if !ok {
		return
	}

	context := event.Context()
	ib := event.Value(P_PROCESS_NAME)
	retExcp := TrimQuotedString(state.call.Value(RET_EXCP))
	sql := state.db.Value(SQL)
	digest := a.normalizer.Digest(sql, state.db.Name())

	a.insertDblock(ib, retExcp, digest, context)
	delete(a.dblockUserEvent, key)
}

func (a *Analysis) handleTdeadlock(event *EventLog) {
	ib := event.Value(P_PROCESS_NAME)
	dims := event.TdeadlockDims()
	context := event.Context()

	a.Tdeadlock.Insert([]string{ib, dims, context}, agg.UpdateContext{Count: 1})
}

func (a *Analysis) handleTlock(key ThreadSessionKey, event *EventLog) {
	eventTtimeout, ok := a.ttimeoutUsrEvent[key]
	if !ok {
		return
	}

	ib := eventTtimeout.Value(P_PROCESS_NAME)
	regions := event.Value(REGIONS)
	context := event.Context()

	if context != "" {
		a.Ttimeout.Insert([]string{ib, regions, context}, agg.UpdateContext{Count: 1})
		delete(a.ttimeoutUsrEvent, key)
	} else {
		a.ttimeoutUsrEvent[key] = event
	}
}

func (a *Analysis) handleDbQuery(key ThreadSessionKey, event *EventLog) {
	context := event.Context()
	if context != "" {
		sql := event.Value(SQL)
		if sql != "" {
			ib := event.Value(P_PROCESS_NAME)
			digest := a.normalizer.Digest(sql, event.Name())
			duration := event.Duration()
			rows := event.Int64Or(ROWS, 0)
			rows += event.Int64Or(ROWS_AFFECTED, 0)

			a.DbQuery.Insert(
				[]string{ib, digest, context},
				agg.UpdateContext{
					Count:    1,
					Duration: duration,
					Rows:     rows,
				},
			)
		}
	}

	if state, ok := a.dblockUserEvent[key]; ok {
		state.db = event
	}
}

func (a *Analysis) insertException(ib, descr, context string) {
	a.Excp.Insert([]string{ib, descr, context}, agg.UpdateContext{Count: 1})
}

func (a *Analysis) insertCall(ib, context string, src *EventLog) {
	duration := src.Duration()
	memoryPeak := src.Int64Or(MEMORY_PEAK, 0)
	cpuTime := src.Int64Or(CPU_TIME, 0)

	a.Call.Insert(
		[]string{ib, context},
		agg.UpdateContext{
			Count:      1,
			Duration:   duration,
			MemoryPeak: memoryPeak,
			CpuTime:    cpuTime,
		},
	)
}

func (a *Analysis) insertDblock(ib, retExcp, digest, context string) {
	a.Dblock.Insert([]string{ib, retExcp, digest, context}, agg.UpdateContext{Count: 1})
}

func (a *Analysis) finalize() {
	a.Call.Calculate()
	a.DbQuery.Calculate()
}
