package onectechcommon

type stackEscapedString []byte

func (s *stackEscapedString) Toggle(v byte) {
	if len(*s) == 0 {
		*s = append(*s, v)
	} else {
		maxIdx := len(*s) - 1
		if (*s)[maxIdx] == v {
			*s = (*s)[:maxIdx]
		} else {
			*s = append(*s, v)
		}
	}
}

func (s *stackEscapedString) IsEmpty() bool {
	return len(*s) == 0
}
