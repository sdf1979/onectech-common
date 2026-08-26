package onectechcommon

import (
	"strconv"
)

type storage struct {
	value uint64
}

func (s *storage) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatUint(s.get(), 10)), nil
}

func (s *storage) add(value uint64) {
	s.value += value
}

func (s *storage) get() uint64 {
	return s.value
}
