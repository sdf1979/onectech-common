package onectechcommon

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
)

type EventLogAggregator struct {
	mu *sync.RWMutex
	m  *metrics
}

func NewEventLogAggregator() *EventLogAggregator {
	return &EventLogAggregator{
		mu: &sync.RWMutex{},
		m:  newMetrics(),
	}
}

func (ela *EventLogAggregator) AddEventLog(el *EventLog) {
	ela.mu.Lock()
	defer ela.mu.Unlock()

	switch el.Name() {
	case EXCP:
		ela.addExcp(el)
	case CALL:
		ela.addCall(el)
	case TTIMEOUT:
		ela.addTtimeout(el)
	case TDEADLOCK:
		ela.addTdeadlock(el)
	}
}

func (ela EventLogAggregator) String() string {
	ela.mu.RLock()
	defer ela.mu.RUnlock()

	b, err := ela.toJSON()
	if err != nil {
		return ""
	}

	return string(b)
}

func (ela *EventLogAggregator) GetValuesAndClear() string {
	ela.mu.Lock()
	defer ela.mu.Unlock()

	var values string

	b, err := ela.toJSON()
	if err != nil {
		values = ""
	}
	values = string(b)
	ela.clear()

	return values
}

func (ela *EventLogAggregator) Clear() {
	ela.mu.Lock()
	defer ela.mu.Unlock()

	ela.clear()
}

func (ela *EventLogAggregator) addExcp(el *EventLog) {
	exception := el.Value(EXCEPTION)
	if exception == DATABASE_EXCEPTION && (strings.Contains(el.Value(DESCR), DB_LOCK_MSSQL) || strings.Contains(el.Value(DESCR), DB_LOCK_POSTGRS)) {
		ela.m.add(ela.m.Excp, "Total", 1)
		ela.m.add(ela.m.Excp, el.Value(P_PROCESS_NAME), 1)

		ela.m.add(ela.m.DbLock, "Total", 1)
		ela.m.add(ela.m.DbLock, el.Value(P_PROCESS_NAME), 1)
	} else if exception == BAD_ALLOC {
		ela.m.add(ela.m.Excp, "Total", 1)
		ela.m.add(ela.m.Excp, el.Value(P_PROCESS_NAME), 1)

		ela.m.add(ela.m.BadAlloc, "Total", 1)
		ela.m.add(ela.m.BadAlloc, el.Value(P_PROCESS_NAME), 1)
	}
}

func (ela *EventLogAggregator) addCall(el *EventLog) {

	counterCall := ela.counterCall(el)

	ela.m.add(ela.m.CallCount, "Total", 1)
	ela.m.add(ela.m.CallCount, counterCall, 1)

	ela.m.add(ela.m.CallDuration, "Total", el.Duration())
	ela.m.add(ela.m.CallDuration, counterCall, el.Duration())

	if cpu, err := strconv.ParseInt(el.Value(CPU_TIME), 10, 64); err == nil {
		ela.m.add(ela.m.CallCpu, "Total", cpu)
		ela.m.add(ela.m.CallCpu, counterCall, cpu)
	}

	if memoryPeak, err := strconv.ParseInt(el.Value(MEMORY_PEAK), 10, 64); err == nil {
		ela.m.add(ela.m.CallMemoryPeak, "Total", memoryPeak)
		ela.m.add(ela.m.CallMemoryPeak, counterCall, memoryPeak)
	}

	if inBytes, err := strconv.ParseInt(el.Value(IN_BYTES), 10, 64); err == nil {
		ela.m.add(ela.m.CallInBytes, "Total", inBytes)
		ela.m.add(ela.m.CallInBytes, counterCall, inBytes)
	}

	if outBytes, err := strconv.ParseInt(el.Value(OUT_BYTES), 10, 64); err == nil {
		ela.m.add(ela.m.CallOutBytes, "Total", outBytes)
		ela.m.add(ela.m.CallOutBytes, counterCall, outBytes)
	}
}

func (ela *EventLogAggregator) addTtimeout(el *EventLog) {
	ela.m.add(ela.m.Excp, "Total", 1)
	ela.m.add(ela.m.Excp, el.Value(P_PROCESS_NAME), 1)

	ela.m.add(ela.m.TTimeout, "Total", 1)
	ela.m.add(ela.m.TTimeout, el.Value(P_PROCESS_NAME), 1)
}

func (ela *EventLogAggregator) addTdeadlock(el *EventLog) {
	ela.m.add(ela.m.Excp, "Total", 1)
	ela.m.add(ela.m.Excp, el.Value(P_PROCESS_NAME), 1)

	ela.m.add(ela.m.TDeadlock, "Total", 1)
	ela.m.add(ela.m.TDeadlock, el.Value(P_PROCESS_NAME), 1)
}

func (ela *EventLogAggregator) counterCall(el *EventLog) string {
	process := el.Value(PROCESS)

	if process != RPHOST {
		return process
	}

	pProcessName := el.Value(P_PROCESS_NAME)
	switch pProcessName {
	case "", SERVER_JOB_EXECUTOR_CONTEXT, DEBUG_CONTROL_CENTER, ADMIN_PROCESS:
		return process
	default:
		return pProcessName
	}
}

func (ela *EventLogAggregator) toJSON() ([]byte, error) {
	return json.Marshal(ela.m)
}

func (ela *EventLogAggregator) clear() {
	ela.m.clear()
}
