package onectechcommon

import (
	"errors"
	"math"

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
	for i := range len(base64Alphabet) {
		t[base64Alphabet[i]] = int8(i)
	}
	return t
}()

// Hash — 64-битный хеш строки.
type Hash uint64

// HashOf вычисляет хеш строки s.
func HashOf(s string) Hash {
	return Hash(xxhash.Sum64String(s))
}

// ParseHash разбирает строковое представление хеша по основанию 64.
// Возвращает ошибку при пустом вводе, недопустимом символе или переполнении.
func ParseHash(text string) (Hash, error) {
	if len(text) == 0 {
		return 0, errors.New("empty string")
	}

	var result uint64
	for i := 0; i < len(text); i++ {
		c := text[i]
		val := base64Lookup[c]
		if val < 0 {
			return 0, errors.New("invalid character: " + string(c))
		}

		// Проверяем переполнение до умножения и сложения.
		if result > (math.MaxUint64-uint64(val))/64 {
			return 0, errors.New("uint64 overflow")
		}
		result = result*64 + uint64(val)
	}
	return Hash(result), nil
}

// String возвращает запись хеша по основанию 64.
func (h Hash) String() string {
	if h == 0 {
		return "A"
	}
	// Для записи uint64 по основанию 64 достаточно 11 цифр.
	buf := make([]byte, 0, 11)
	for h > 0 {
		buf = append(buf, base64Alphabet[h%64])
		h /= 64
	}
	// Разворачиваем цифры, собранные от младшей к старшей.
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
