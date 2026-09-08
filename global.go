package onectechcommon

import (
	"strconv"
	"strings"
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
	builder.Grow(len(s)) // выделяем память под результирующую строку (не больше исходной)

	n := len(s)
	plen := len(prefix)
	i := 0

	for i < n {
		// Проверяем, начинается ли подстрока с prefix
		if i+plen <= n && s[i:i+plen] == prefix {
			// Если сразу после prefix стоит цифра – начинаем замену
			if i+plen < n && s[i+plen] >= '0' && s[i+plen] <= '9' {
				// Пропускаем все цифры
				j := i + plen
				for j < n && s[j] >= '0' && s[j] <= '9' {
					j++
				}

				// Записываем префикс без цифр
				builder.WriteString(prefix)

				// Если после цифр идёт пробел – копируем его,
				// чтобы не нарушить форматирование
				if j < n && s[j] == ' ' {
					builder.WriteByte(' ')
					i = j + 1
				} else {
					i = j
				}
				continue
			}
		}

		// Если совпадения нет – копируем текущий символ
		builder.WriteByte(s[i])
		i++
	}

	return builder.String()
}

func IsInteger(s string) bool {
	s = strings.TrimSpace(s)
	_, err := strconv.Atoi(s)
	return err == nil
}
