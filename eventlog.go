package onectechcommon

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

type keyValue struct {
	Name string
	Idx  int
}

type EventLog struct {
	tm           time.Time
	name         string
	duration     int64
	nestingLevel int16
	keyOrder     []keyValue
	values       map[string][]string
}

func NewEvent(t time.Time, data []byte) *EventLog {
	//Time
	e := &EventLog{}
	tm, _ := time.Parse("04:05.000000", string(data[:12]))
	e.tm = time.Date(t.Year(), t.Month(), t.Day(),
		t.Hour(), tm.Minute(), tm.Second(), tm.Nanosecond(),
		t.Location())
	data = data[13:]

	//Duration
	idx := bytes.IndexByte(data, ',')
	if idx == -1 {
		return e
	}
	e.duration = asciiToInt64(data[:idx])
	data = data[idx+1:]

	//Name
	idx = bytes.IndexByte(data, ',')
	if idx == -1 {
		return e
	}
	e.name = string(data[:idx])
	data = data[idx+1:]

	//Nesting level
	idx = bytes.IndexByte(data, ',')
	if idx == -1 {
		return e
	}
	e.nestingLevel = asciiToInt16(data[:idx])
	data = data[idx+1:]

	e.values = make(map[string][]string)

	for {
		idx = bytes.IndexByte(data, '=')
		if idx == -1 {
			break
		}
		key := string(data[:idx])
		data = data[idx+1:]

		if len(data) > 0 && (data[0] == '\'' || data[0] == '"') {
			escChar := data[0]
			idx = findInEscapedString(data, escChar)
		} else {
			idx = bytes.IndexByte(data, ',')
		}
		if idx == -1 {
			idx = bytes.IndexByte(data, '\r')
			if idx == -1 {
				idx = bytes.IndexByte(data, '\n')
			}
			if idx == -1 {
				break
			}
		}
		value := e.values[key]
		e.keyOrder = append(e.keyOrder, keyValue{key, len(value)})
		value = append(value, string(data[:idx]))
		e.values[key] = value
		data = data[idx+1:]
	}

	return e
}

func (e *EventLog) Name() string {
	return e.name
}

func (e *EventLog) Duration() int64 {
	return e.duration
}

func (e *EventLog) NestingLevel() int16 {
	return e.nestingLevel
}

func (e *EventLog) Value(key string) string {
	v := e.values[key]
	if v == nil {
		return ""
	}

	if key == P_PROCESS_NAME {
		return v[0]
	}

	return v[len(v)-1]
}

func (e *EventLog) IsDbLock() bool {
	if e.name == "EXCP" && e.Value("Exception") == "DataBaseException" && strings.Contains(e.Value("Descr"), "Lock request time out period exceeded") {
		return true
	}
	return false
}

func (e EventLog) String() string {
	var sb strings.Builder

	sb.WriteString(e.tm.Format("06010215:04:05.000000"))
	sb.WriteString("-")
	sb.WriteString(strconv.FormatInt(e.duration, 10))
	sb.WriteString(",")
	sb.WriteString(e.name)
	sb.WriteString(",")
	sb.WriteString(strconv.FormatInt(int64(e.nestingLevel), 10))

	for _, key := range e.keyOrder {
		sb.WriteString(",")
		sb.WriteString(key.Name)
		sb.WriteString("=")
		sb.WriteString(e.values[key.Name][key.Idx])
	}

	return sb.String()
}

func asciiToInt64(data []byte) int64 {
	var n int64
	for _, ch := range data {
		n = n*10 + int64(ch-'0')
	}
	return n
}

func asciiToInt16(data []byte) int16 {
	var n int16
	for _, ch := range data {
		n = n*10 + int16(ch-'0')
	}
	return n
}

func findInEscapedString(data []byte, v byte) int {
	stack := stackEscapedString{}
	for idx := 0; idx < len(data); idx++ {
		if data[idx] == v {
			stack.Toggle(v)
		} else if (data[idx] == ',' || data[idx] == '\r' || data[idx] == '\n') && stack.IsEmpty() {
			return idx
		}
	}
	return -1
}
