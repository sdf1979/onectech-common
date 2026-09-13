package onectechcommon

import (
	"sort"

	agg "github.com/sdf1979/onectech-common/aggregator"
)

// Node — узел дерева агрегации. Каждый узел хранит свой ключ
// (строку, под которой он лежит в children родителя), агрегатор
// и карту детей. Корень дерева имеет key == "".
type Node struct {
	key      string
	children map[string]*Node
	agg      agg.Aggregator
}

// NewNode создаёт корневой узел с заданным агрегатором.
func NewNode(a agg.Aggregator) *Node {
	return &Node{
		children: make(map[string]*Node),
		agg:      a,
	}
}

// Insert добавляет путь в дерево, применяя ctx к агрегатору
// на каждом уровне. Отсутствующие узлы создаются на лету,
// их агрегаторы клонируются от родителя.
func (n *Node) Insert(path []string, ctx agg.UpdateContext) {
	current := n
	for _, key := range path {
		current.agg.Apply(ctx)
		child, ok := current.children[key]
		if !ok {
			child = NewNode(current.agg.Clone())
			child.key = key
			current.children[key] = child
		}
		current = child
	}
	current.agg.Apply(ctx)
}

// Calculate пересчитывает агрегаторы всего дерева сверху вниз.
func (n *Node) Calculate() {
	n.calculateRecursive(nil)
}

func (n *Node) calculateRecursive(parent *Node) {
	var parentAgg agg.Aggregator
	if parent != nil {
		parentAgg = parent.agg
	}
	n.agg.Calculate(parentAgg)
	for _, c := range n.children {
		c.calculateRecursive(n)
	}
}

// Key возвращает ключ узла — строку, под которой он лежит
// в children родителя. Для корня возвращает "".
func (n *Node) Key() string { return n.key }

// Agg даёт доступ к агрегатору узла.
func (n *Node) Agg() agg.Aggregator { return n.agg }

// Hash возвращает Base64-представление xxhash от ключа узла.
// Для корня (key == "") возвращает "" — корень не имеет
// собственного ключа, и хешировать нечего.
func (n *Node) Hash() string {
	if n.key == "" {
		return ""
	}
	return Uint64ToBase64(HashString([]byte(n.key)))
}

// Walk обходит дерево в pre-order: сначала узел, затем дети
// в порядке убывания Score (см. sortedChildren).
// Порядок обхода — часть контракта: на него опирается htmltable,
// который строит строки таблицы в порядке вызова.
func (n *Node) Walk(fn func(node *Node, depth int)) {
	var rec func(node *Node, depth int)
	rec = func(node *Node, depth int) {
		fn(node, depth)
		for _, k := range sortedChildren(node.children) {
			rec(node.children[k], depth+1)
		}
	}
	rec(n, 0)
}

// sortedChildren возвращает ключи детей в порядке убывания Score,
// при равенстве — по алфавиту.
func sortedChildren(m map[string]*Node) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]].agg.Score() != m[keys[j]].agg.Score() {
			return m[keys[i]].agg.Score() > m[keys[j]].agg.Score()
		}
		return keys[i] < keys[j]
	})
	return keys
}
