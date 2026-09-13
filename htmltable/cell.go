package htmltable

import (
	"html"
	"strconv"
	"strings"
)

// MaxCellChars — сколько символов показывать в ячейке напрямую.
// Полный текст кладётся в скрытый <span class="oct-text-full">
// и разворачивается по клику.
const MaxCellChars = 500

// writeCell пишет содержимое ячейки. Длинные значения усекаются
// по границе UTF-8, полный текст кладётся рядом скрытым и
// разворачивается по клику.
func writeCell(sb *strings.Builder, v string) {
	if len(v) <= MaxCellChars {
		sb.WriteString(`<div class="oct-cell">`)
		sb.WriteString(html.EscapeString(v))
		sb.WriteString(`</div>`)
		return
	}
	// Оба текста лежат в DOM одновременно: короткий виден, полный — скрыт.
	// Клик лишь переключает класс oct-open у родителя.
	sb.WriteString(`<div class="oct-cell oct-clip">`)
	sb.WriteString(`<span class="oct-toggle"></span>`)
	sb.WriteString(`<span class="oct-text-short">`)
	sb.WriteString(html.EscapeString(truncateUTF8(v, MaxCellChars)))
	sb.WriteString(`…</span>`)
	sb.WriteString(`<span class="oct-text-full">`)
	sb.WriteString(html.EscapeString(v))
	sb.WriteString(`</span>`)
	sb.WriteString(`</div>`)
}

// truncateUTF8 режет строку по байтам, не разрывая UTF-8 руны.
func truncateUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n]
}

// numericKey нормализует значение: если это число — канонический вид
// для клиентской числовой сортировки, иначе — исходную строку.
func numericKey(v string) string {
	if v == "" {
		return ""
	}
	c := v[0]
	if !(c >= '0' && c <= '9') && c != '-' && c != '+' && c != '.' {
		return v
	}
	if f, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 64); err == nil {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return v
}
