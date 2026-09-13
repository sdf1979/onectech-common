package onectechcommon

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
)

// Base64-алфавит для системы счисления:
// A-Z = 0..25, a-z = 26..51, 0-9 = 52..61, '+' = 62, '/' = 63
const base64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// lookup для быстрого преобразования символа в значение.
// Инициализируется один раз при загрузке пакета.
var base64Lookup = func() [256]int8 {
	var t [256]int8
	for i := range t {
		t[i] = -1
	}
	for i := 0; i < len(base64Alphabet); i++ {
		t[base64Alphabet[i]] = int8(i)
	}
	return t
}()

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

func HashString(data []byte) uint64 {
	return xxhash.Sum64(data)
}

func Uint64ToBase64(n uint64) string {
	if n == 0 {
		return "A"
	}
	// Максимальное количество цифр для uint64 в Base64: ceil(64 / log2(64)) = 11
	buf := make([]byte, 0, 11)
	for n > 0 {
		rem := n % 64
		buf = append(buf, base64Alphabet[rem])
		n /= 64
	}
	// Разворачиваем, так как цифры собирались с конца
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// Base64ToUint64 преобразует строку в системе счисления Base64 в uint64.
// Возвращает ошибку при недопустимом символе или переполнении.
func Base64ToUint64(s string) (uint64, error) {
	if len(s) == 0 {
		return 0, errors.New("empty string")
	}

	var result uint64
	maxUint64 := uint64(math.MaxUint64)

	for i := 0; i < len(s); i++ {
		c := s[i]
		val := base64Lookup[c]
		if val < 0 {
			return 0, errors.New("invalid character: " + string(c))
		}

		// Проверка переполнения перед умножением и сложением:
		// result*64 + val > maxUint64  <=>  result > (maxUint64 - val) / 64
		if result > (maxUint64-uint64(val))/64 {
			return 0, errors.New("uint64 overflow")
		}
		result = result*64 + uint64(val)
	}
	return result, nil
}

func ContainsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
