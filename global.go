package onectechcommon

import (
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
)

func TrimQuotedString(s string) string {
	if len(s) < 2 {
		return s
	}
	first := s[0]
	last := s[len(s)-1]
	if (first == '\'' || first == '"') && first == last {
		start := 1
		end := len(s) - 1

		// Удаляем все \r и \n сразу после открывающей кавычки
		for start < end && (s[start] == '\r' || s[start] == '\n') {
			start++
		}
		// Удаляем все \r и \n перед закрывающей кавычкой
		for end > start && (s[end-1] == '\r' || s[end-1] == '\n') {
			end--
		}

		if start > end {
			return ""
		}
		// Срез без копирования данных
		return s[start:end]
	}
	return s
}

func GetNextWord(s, word string) string {
	pos := strings.Index(s, word)
	s = s[pos+len(word)+1:]
	pos = strings.Index(s, " ")
	s = s[0:pos]
	return s
}

func ReplacePrefixDigits(s string, prefix string) string {
	var builder strings.Builder
	copyFrom := 0
	searchFrom := 0

	for searchFrom < len(s) {
		match := strings.Index(s[searchFrom:], prefix)
		if match < 0 {
			break
		}
		match += searchFrom

		digitStart := match + len(prefix)
		if digitStart == len(s) {
			break
		}
		if s[digitStart] < '0' || s[digitStart] > '9' {
			searchFrom = match + 1
			continue
		}

		digitEnd := digitStart + 1
		for digitEnd < len(s) && s[digitEnd] >= '0' && s[digitEnd] <= '9' {
			digitEnd++
		}

		if copyFrom == 0 {
			builder.Grow(len(s))
		}
		builder.WriteString(s[copyFrom:digitStart])
		copyFrom = digitEnd
		searchFrom = digitEnd

		// Как и в исходной функции, сохраняем пробел после цифр
		// и не используем его как начало следующего совпадения.
		if searchFrom < len(s) && s[searchFrom] == ' ' {
			searchFrom++
		}
	}

	if copyFrom == 0 {
		return s
	}
	builder.WriteString(s[copyFrom:])
	return builder.String()
}

func IsInteger(s string) bool {
	s = strings.TrimSpace(s)
	_, err := strconv.Atoi(s)
	return err == nil
}

// HashString вычисляет 64-битный хеш последовательности байтов.
//
// Deprecated: используйте [HashOf] для вычисления хеша строки.
func HashString(data []byte) uint64 {
	return xxhash.Sum64(data)
}

// Uint64ToBase64 возвращает запись числа по основанию 64.
//
// Deprecated: используйте метод [Hash.String].
func Uint64ToBase64(n uint64) string {
	return Hash(n).String()
}

// Base64ToUint64 преобразует строку в системе счисления Base64 в uint64.
// Возвращает ошибку при недопустимом символе или переполнении.
//
// Deprecated: используйте [ParseHash].
func Base64ToUint64(s string) (uint64, error) {
	h, err := ParseHash(s)
	return uint64(h), err
}

func ContainsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
