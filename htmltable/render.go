package htmltable

import (
	"fmt"
	"html"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"

	oct "github.com/sdf1979/onectech-common"
	agg "github.com/sdf1979/onectech-common/aggregator"
)

// renderSeq — счётчик вызовов Render в процессе. Гарантирует
// уникальные id чекбоксов, когда на одной странице несколько таблиц.
var renderSeq uint64

// Options — параметры рендера HTML-таблицы.
type Options struct {
	FirstColumn string // заголовок первой колонки, обычно "Итого:"
	SortColumn  string // колонка стартовой сортировки; "" → "Score"
	HashLevel   int    // с какого depth выводить хеш
}

// Render строит готовую HTML-таблицу из дерева агрегации.
// Никакого JSON и JS-рендера: JS только сортирует готовые <tr>.
//
// Колонка "Хеш" выводится, начиная с глубины opts.HashLevel.
// Хеш берётся у самого узла (Node.Hash), htmltable не знает,
// каким алгоритмом он получен.
func Render(root *oct.Node, opts Options) string {
	headers, getters, scoreCol, sortDir := buildColumns(root.Agg(), opts)

	// Уникальный id для связки «чекбокс ↔ таблица» на странице.
	seq := atomic.AddUint64(&renderSeq, 1)
	filterID := "oct-filter-" + strconv.FormatUint(seq, 10)

	var sb strings.Builder
	sb.Grow(1 << 20) // стартовый буфер 1 МБ — не реаллоцируем постоянно

	sb.WriteString(`<style>`)
	sb.WriteString(tableCSS)
	sb.WriteString(`</style>`)

	// Флажок ограничения выборки. По умолчанию включён.
	// Связь с таблицей — через id: table[data-filter] = input#id.
	sb.WriteString(`<div class="oct-controls">`)
	sb.WriteString(`<label class="oct-filter" for="`)
	sb.WriteString(filterID)
	sb.WriteString(`"><input type="checkbox" id="`)
	sb.WriteString(filterID)
	sb.WriteString(`" class="oct-filter-toggle" checked>`)
	sb.WriteString(`<span class="oct-filter-text">Ограничить выборку данных</span>`)
	sb.WriteString(`</label>`)
	sb.WriteString(`</div>`)

	sb.WriteString(`<table class="oct-table" data-filter="`)
	sb.WriteString(filterID)
	sb.WriteString(`"`)
	if scoreCol > 0 {
		fmt.Fprintf(&sb, ` data-sort-col="%d" data-sort-dir="%d"`, scoreCol, sortDir)
	}
	sb.WriteString(`><thead><tr>`)

	writeHeaderRow(&sb, headers, scoreCol, sortDir)
	sb.WriteString(`</tr></thead><tbody>`)

	// Walk идёт pre-order, поэтому на момент отрисовки строки
	// глубины d строка-родитель (depth-1) уже отрисована.
	// Достаточно хранить id последней строки на каждой глубине.
	rowID := 0
	parentByDepth := make([]string, 0, 8)

	root.Walk(func(node *oct.Node, depth int) {
		key := node.Key()
		if depth == 0 {
			key = opts.FirstColumn
		}

		// Хеш берём у самого узла — htmltable не знает,
		// что там xxhash и Base64.
		hashStr := ""
		if depth >= opts.HashLevel {
			hashStr = node.Hash()
		}

		id := strconv.Itoa(rowID)
		rowID++

		parentID := ""
		if depth > 0 && depth <= len(parentByDepth) {
			parentID = parentByDepth[depth-1]
		}
		writeRow(&sb, id, parentID, depth, key, hashStr, node.Agg(), getters)

		if depth < len(parentByDepth) {
			parentByDepth[depth] = id
		} else {
			parentByDepth = append(parentByDepth, id)
		}
	})

	sb.WriteString(`</tbody></table>`)
	sb.WriteString(`<script>`)
	sb.WriteString(tableJS)
	sb.WriteString(`</script>`)
	return sb.String()
}

// buildColumns инспектирует тип агрегатора и возвращает:
// заголовки, геттеры значений, индекс колонки Score и направление сортировки.
func buildColumns(a agg.Aggregator, opts Options) (
	headers []string,
	getters []func(agg.Aggregator) string,
	scoreCol int,
	sortDir int,
) {
	scoreCol = -1
	sortDir = -1

	aggType := reflect.TypeOf(a)
	if aggType.Kind() == reflect.Pointer {
		aggType = aggType.Elem()
	}

	if aggType.Kind() != reflect.Struct {
		// Не-структура: колонка "Хеш" + "Score".
		headers = []string{opts.FirstColumn, "Хеш", "Score"}
		getters = []func(agg.Aggregator) string{
			func(a agg.Aggregator) string { return strconv.Itoa(int(a.Score())) },
		}
		return headers, getters, 2, -1
	}

	headers = []string{opts.FirstColumn, "Хеш"}
	headerIdx := 2

	for i := 0; i < aggType.NumField(); i++ {
		f := aggType.Field(i)
		if f.PkgPath != "" {
			continue
		}
		col := f.Name
		if tag, ok := f.Tag.Lookup("json"); ok {
			parts := strings.Split(tag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				col = parts[0]
			}
		}
		headers = append(headers, col)

		if isScoreField(f.Name, col, opts.SortColumn) {
			scoreCol = headerIdx
		}
		headerIdx++

		idx := i
		getters = append(getters, func(a agg.Aggregator) string {
			v := reflect.ValueOf(a)
			if v.Kind() == reflect.Pointer {
				v = v.Elem()
			}
			return fmt.Sprintf("%v", v.Field(idx).Interface())
		})
	}
	return headers, getters, scoreCol, sortDir
}

// isScoreField: приоритет — явно заданное имя > имя поля "Score" > json-тег "score".
func isScoreField(fieldName, col, sortColumnName string) bool {
	switch {
	case sortColumnName != "":
		return col == sortColumnName || fieldName == sortColumnName
	case fieldName == "Score" || strings.EqualFold(col, "score"):
		return true
	}
	return false
}

// writeHeaderRow рендерит <thead><tr>. Первые две колонки (ключ и хеш)
// не сортируемые; остальные получают стрелку. Стрелка рендерится
// во всех сортируемых заголовках, неактивные прячутся через
// visibility:hidden, чтобы ширина <th> не менялась при смене колонки.
func writeHeaderRow(sb *strings.Builder, headers []string, scoreCol, sortDir int) {
	for i, h := range headers {
		if i == 0 || i == 1 {
			sb.WriteString(`<th class="oct-th">`)
			sb.WriteString(html.EscapeString(h))
			sb.WriteString(`</th>`)
			continue
		}
		sb.WriteString(`<th class="oct-th oct-sortable" data-col="`)
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(`">`)
		sb.WriteString(html.EscapeString(h))

		arrow := "▼"
		if sortDir == 1 {
			arrow = "▲"
		}
		if i == scoreCol {
			sb.WriteString(`<span class="oct-arrow">`)
		} else {
			sb.WriteString(`<span class="oct-arrow oct-arrow-off">`)
		}
		sb.WriteString(arrow)
		sb.WriteString(`</span>`)

		sb.WriteString(`</th>`)
	}
}

// writeRow рендерит одну <tr>. hashStr приходит готовым —
// хеш посчитан на уровне Node, а не здесь.
func writeRow(
	sb *strings.Builder,
	id, parentID string,
	depth int,
	key, hashStr string,
	a agg.Aggregator,
	getters []func(agg.Aggregator) string,
) {
	rowClass := "oct-d0"
	if depth%2 != 0 {
		rowClass = "oct-d1"
	}
	sb.WriteString(`<tr class="`)
	sb.WriteString(rowClass)
	sb.WriteString(`" data-id="`)
	sb.WriteString(id)
	sb.WriteString(`" data-parent="`)
	sb.WriteString(parentID)
	sb.WriteString(`">`)

	// Первая колонка — ключ с отступом по глубине.
	sb.WriteString(`<td class="oct-td-key" style="padding-left:`)
	sb.WriteString(strconv.Itoa(depth * 20))
	sb.WriteString(`px">`)
	if depth == 0 {
		writeCell(sb, " Итого")
	} else {
		writeCell(sb, key)
	}
	sb.WriteString(`</td>`)

	// Вторая колонка — Хеш (Base64). Для строк выше hashLevel
	// hashStr == "", ячейка остаётся пустой.
	sb.WriteString(`<td class="oct-td oct-td-hash">`)
	sb.WriteString(html.EscapeString(hashStr))
	sb.WriteString(`</td>`)

	// Колонки агрегатора — data-k для клиентской сортировки.
	// oct-td-nw запрещает перенос строк в этих ячейках.
	for _, g := range getters {
		v := g(a)
		sb.WriteString(`<td class="oct-td oct-td-nw" data-k="`)
		sb.WriteString(html.EscapeString(numericKey(v)))
		sb.WriteString(`">`)
		writeCell(sb, v)
		sb.WriteString(`</td>`)
	}
	sb.WriteString(`</tr>`)
}
