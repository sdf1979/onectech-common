package onectechcommon

import (
	"strconv"
	"time"
)

type storage struct {
	value      int64
	lastAccess time.Time
}

func (s *storage) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(s.get(), 10)), nil
}

func (s *storage) add(value int64) {
	s.value += value
	if s.lastAccess.IsZero() {
		s.lastAccess = time.Now()
	}
}

func (s *storage) get() int64 {
	if time.Since(s.lastAccess).Seconds() > 300 {
		s.value = 0
	}
	s.lastAccess = time.Now()
	return s.value
}

func (s *storage) clear() {
	s.value = 0
	s.lastAccess = time.Now()
}
