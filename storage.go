package onectechcommon

import (
	"strconv"
)

type storage struct {
	value int64
}

func (s *storage) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(s.get(), 10)), nil
}

func (s *storage) add(value int64) {
	s.value += value
}

func (s *storage) get() int64 {
	return s.value
}
