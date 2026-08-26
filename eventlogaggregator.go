package onectechcommon

import (
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

func (ela *EventLogAggregator) GetMetric(key, param string) uint64 {
	ela.mu.Lock()
	defer ela.mu.Unlock()

	value := ela.m.getValue(key, param)

	return value
}

func (ela *EventLogAggregator) GetMetrics() string {
	ela.mu.Lock()
	defer ela.mu.Unlock()

	var values string

	b, err := ela.toJSON()
	if err != nil {
		values = ""
	}
	values = string(b)

	return values
}

func (ela *EventLogAggregator) GetMetricsLLD() string {
	ela.mu.Lock()
	defer ela.mu.Unlock()

	var values string

	b, err := ela.toLLD()
	if err != nil {
		values = ""
	}
	values = string(b)

	return values
}

func (ela *EventLogAggregator) GetKeys() []string {
	ela.mu.Lock()
	defer ela.mu.Unlock()

	return ela.m.getKeys()
}

func (ela *EventLogAggregator) addExcp(el *EventLog) {
	switch el.Value(EXCEPTION) {
	case DATABASE_EXCEPTION:
		descr := el.Value(DESCR)
		if strings.Contains(descr, DB_LOCK_MSSQL) || strings.Contains(descr, DB_LOCK_POSTGRS) {
			pProcessName := el.Value(P_PROCESS_NAME)
			ela.m.add(ela.m.Excp, "Total", 1)
			ela.m.add(ela.m.Excp, pProcessName, 1)

			ela.m.add(ela.m.DbLock, "Total", 1)
			ela.m.add(ela.m.DbLock, pProcessName, 1)
		} else if strings.Contains(descr, DB_DEADLOCK_MSSQL) {
			pProcessName := el.Value(P_PROCESS_NAME)
			ela.m.add(ela.m.Excp, "Total", 1)
			ela.m.add(ela.m.Excp, pProcessName, 1)

			ela.m.add(ela.m.DbDeadlock, "Total", 1)
			ela.m.add(ela.m.DbDeadlock, pProcessName, 1)
		} else if strings.Contains(descr, CANNOT_INSERT_DUPLICATE_KEY_MSSQL) || strings.Contains(descr, TRANSACTION_ALREADY_HAS_ERRORS_RU) {
			ela.m.add(ela.m.Excp, "Total", 1)
			ela.m.add(ela.m.Excp, el.Value(P_PROCESS_NAME), 1)
		}
	case BAD_ALLOC:
		pProcessName := el.Value(P_PROCESS_NAME)
		ela.m.add(ela.m.Excp, "Total", 1)
		ela.m.add(ela.m.Excp, pProcessName, 1)

		ela.m.add(ela.m.BadAlloc, "Total", 1)
		ela.m.add(ela.m.BadAlloc, pProcessName, 1)
	default:
		if el.Value(CONTEXT) != "" {
			descr := el.Value(DESCR)
			if strings.Contains(descr, FORM_UNAVAILABLE_FOR_USE_RU) {
				ela.m.add(ela.m.Excp, "Total", 1)
				ela.m.add(ela.m.Excp, el.Value(P_PROCESS_NAME), 1)
			}
		}
	}
}

func (ela *EventLogAggregator) addCall(el *EventLog) {

	counterCall := ela.counterCall(el)

	ela.m.add(ela.m.CallCount, "Total", 1)
	ela.m.add(ela.m.CallCount, counterCall, 1)

	ela.m.add(ela.m.CallDuration, "Total", uint64(el.Duration()))
	ela.m.add(ela.m.CallDuration, counterCall, uint64(el.Duration()))

	if cpu, err := strconv.ParseUint(el.Value(CPU_TIME), 10, 64); err == nil {
		ela.m.add(ela.m.CallCpu, "Total", cpu)
		ela.m.add(ela.m.CallCpu, counterCall, cpu)
	}

	if memoryPeak, err := strconv.ParseUint(el.Value(MEMORY_PEAK), 10, 64); err == nil {
		ela.m.add(ela.m.CallMemoryPeak, "Total", memoryPeak)
		ela.m.add(ela.m.CallMemoryPeak, counterCall, memoryPeak)
	}

	if inBytes, err := strconv.ParseUint(el.Value(IN_BYTES), 10, 64); err == nil {
		ela.m.add(ela.m.CallInBytes, "Total", inBytes)
		ela.m.add(ela.m.CallInBytes, counterCall, inBytes)
	}

	if outBytes, err := strconv.ParseUint(el.Value(OUT_BYTES), 10, 64); err == nil {
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
	return ela.m.toJSON()
}

func (ela *EventLogAggregator) toLLD() ([]byte, error) {
	return ela.m.toLLD()
}
