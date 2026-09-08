package onectechcommon

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	agg "github.com/sdf1979/onectech-common/aggregator"
)

type Node struct {
	children map[string]*Node
	agg      agg.Aggregator
	descr    string
}

func NewNode(agg agg.Aggregator) *Node {
	return &Node{
		children: make(map[string]*Node),
		agg:      agg,
	}
}

func (n *Node) Insert(path []string, ctx agg.UpdateContext) {
	current := n
	for _, key := range path {
		current.agg.Apply(ctx)
		child, ok := current.children[key]
		if !ok {
			child = NewNode(current.agg.Clone())
			current.children[key] = child
		}
		current = child
	}
	current.agg.Apply(ctx)
}

func (n *Node) Calculate() {
	n.calculateRecursive(nil)
}
func sortedChildren(m map[string]*Node) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]].agg.Score() != m[keys[j]].agg.Score() {
			return m[keys[i]].agg.Score() > m[keys[j]].agg.Score() // убывание
		}
		return keys[i] < keys[j] // при равных count – по алфавиту
	})
	return keys
}

func (n *Node) calculateRecursive(parent *Node) {
	var parentAgg agg.Aggregator
	if parent != nil {
		parentAgg = parent.agg
	}
	n.agg.Calculate(parentAgg)

	for _, child := range n.children {
		child.calculateRecursive(n)
	}
}

func (n *Node) ToHTML(nameFirstColumn string) string {
	aggType := reflect.TypeOf(n.agg)
	if aggType.Kind() == reflect.Ptr {
		aggType = aggType.Elem()
	}

	var headers []string
	var getters []func(agg.Aggregator) string

	if aggType.Kind() == reflect.Struct {
		headers = append(headers, nameFirstColumn)

		for i := 0; i < aggType.NumField(); i++ {
			field := aggType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			colName := field.Name
			if tag, ok := field.Tag.Lookup("json"); ok {
				parts := strings.Split(tag, ",")
				if parts[0] == "-" {
					continue
				}
				if parts[0] != "" {
					colName = parts[0]
				}
			}
			headers = append(headers, colName)

			idx := i
			getters = append(getters, func(a agg.Aggregator) string {
				v := reflect.ValueOf(a)
				if v.Kind() == reflect.Ptr {
					v = v.Elem()
				}
				fieldVal := v.Field(idx)
				return fmt.Sprintf("%v", fieldVal.Interface())
			})
		}
	} else {
		headers = []string{nameFirstColumn, "Score"}
		getters = []func(agg.Aggregator) string{
			func(a agg.Aggregator) string { return fmt.Sprintf("%d", a.Score()) },
		}
	}

	var sb strings.Builder
	sb.WriteString(`<table border="1" cellpadding="5" style="border-collapse: collapse; width: 100%;">`)

	// Заголовок – тёмный фон, белый текст
	sb.WriteString("<thead><tr>")
	for _, h := range headers {
		sb.WriteString(`<th style="vertical-align: top; background-color: #006B9E; color: #fff; padding: 8px;">`)
		sb.WriteString(h)
		sb.WriteString(`</th>`)
	}
	sb.WriteString("</tr></thead><tbody>")

	// Выводим корневой узел с ключом "Итого:" и глубиной 0
	n.tableFormat(&sb, 0, "Итого:", headers, getters)

	sb.WriteString("</tbody></table>")
	return sb.String()
}

func (n *Node) tableFormat(sb *strings.Builder, depth int, key string, headers []string, getters []func(agg.Aggregator) string) {
	const maxHeight = "8.0em"

	// Добавляем CSS-стили один раз (при depth == 0 – корневой вызов)
	if depth == 0 {
		sb.WriteString(`<style>
			.cell-scroll {
				max-height: ` + maxHeight + `;
				overflow-y: hidden;
				white-space: pre-wrap;
				word-break: break-word;
				scrollbar-gutter: stable;
			}
			.cell-scroll:hover {
				overflow-y: auto;
			}
		</style>`)
	}

	// Цвет фона по чётности глубины
	bgColor := ""
	if depth%2 == 0 {
		bgColor = ` style="background-color: #ffffff;"`
	} else {
		bgColor = ` style="background-color: #e9e9e9;"`
	}

	sb.WriteString("<tr")
	sb.WriteString(bgColor)
	sb.WriteString(">")

	// Первая колонка – ключ с отступом
	padding := depth * 20
	fmt.Fprintf(sb, `<td style="vertical-align: top; padding-left:%dpx;">`, padding)
	fmt.Fprintf(sb, `<div class="cell-scroll">%s</div>`, key)
	sb.WriteString("</td>")

	// Остальные колонки – значения
	for _, getter := range getters {
		sb.WriteString(`<td style="vertical-align: top; text-align: right;">`)
		val := getter(n.agg)
		fmt.Fprintf(sb, `<div class="cell-scroll">%s</div>`, val)
		sb.WriteString("</td>")
	}

	sb.WriteString("</tr>")

	// Рекурсивный спуск к детям с увеличением глубины
	childrenKeys := sortedChildren(n.children)
	for _, childKey := range childrenKeys {
		child := n.children[childKey]
		child.tableFormat(sb, depth+1, childKey, headers, getters)
	}
}
