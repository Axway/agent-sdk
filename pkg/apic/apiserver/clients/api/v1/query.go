package v1

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=QueryOp
type QueryOp int

const (
	In QueryOp = iota
	Eq
)

const (
	opIn  = "=in="
	opAnd = ";"
	opOr  = ","
)

type andNode []QueryNode

func (n andNode) Accept(v Visitor) {
	v.Visit(n)
}

type orNode []QueryNode

func (n orNode) Accept(v Visitor) {
	v.Visit(n)
}

type attrNode struct {
	key    string
	values []string
}

type tagNode []string

func (n *attrNode) Accept(v Visitor) {
	v.Visit(n)
}

func (n tagNode) Accept(v Visitor) {
	v.Visit(n)
}

type Visitor interface {
	Visit(QueryNode)
}

type QueryNode interface {
	Accept(Visitor)
}

func AttrIn(key string, values ...string) QueryNode {
	return &attrNode{
		key,
		values,
	}
}

func TagsIn(values ...string) QueryNode {
	return tagNode(values)
}

func AnyAttr(attrs map[string]string) QueryNode {
	nodes := make([]QueryNode, len(attrs))
	i := 0
	for key, val := range attrs {
		nodes[i] = AttrIn(key, val)
		i++
	}
	return orNode(nodes)
}

func AllAttr(attrs map[string]string) QueryNode {
	nodes := make([]QueryNode, len(attrs))
	i := 0
	for key, val := range attrs {
		nodes[i] = AttrIn(key, val)
		i++
	}
	return andNode(nodes)
}

func Or(first, second QueryNode, rest ...QueryNode) QueryNode {
	nodes := make([]QueryNode, len(rest)+2)
	nodes[0] = first
	nodes[1] = second

	return orNode(append(nodes, rest...))
}

func And(first, second QueryNode, rest ...QueryNode) QueryNode {
	nodes := make([]QueryNode, len(rest)+2)
	nodes[0] = first
	nodes[1] = second

	return andNode(append(nodes, rest...))
}

// RSQLVisitor builds an RSQL string by visiting QueryNodes
type RSQLVisitor struct {
	b strings.Builder
}

func NewRSQLVisitor() *RSQLVisitor {
	return &RSQLVisitor{
		b: strings.Builder{},
	}
}

func (rv *RSQLVisitor) String() string {
	return rv.b.String()
}

func (rv *RSQLVisitor) Visit(node QueryNode) {

	switch n := node.(type) {
	case andNode:
		rv.b.WriteString("(")
		children := []QueryNode(n)
		for i, child := range children {
			child.Accept(rv)
			if i < len(children)-1 {
				rv.b.WriteString(";")
			}
		}
		rv.b.WriteString(")")
	case orNode:
		rv.b.WriteString("(")
		children := []QueryNode(n)
		for i, child := range children {
			child.Accept(rv)
			if i < len(children)-1 {
				rv.b.WriteString(",")
			}
		}
		rv.b.WriteString(")")
	case tagNode:
		switch len(n) {
		case 0:
			rv.b.WriteString(`tags==""`)
		case 1:
			rv.b.WriteString(fmt.Sprintf(`tags=="%s"`, n[0]))
		default:
			rv.b.WriteString(fmt.Sprintf(`tags=in=("%s")`, strings.Join(n, `","`)))
		}
	case *attrNode:
		switch len(n.values) {
		case 0:
			rv.b.WriteString(fmt.Sprintf(`attributes.%s==""`, n.key))
		case 1:
			rv.b.WriteString(fmt.Sprintf(`attributes.%s=="%s"`, n.key, n.values[0]))
		default:
			rv.b.WriteString(fmt.Sprintf(`attributes.%s=in=("%s")`, n.key, strings.Join(n.values, `","`)))
		}
	default:
		panic(fmt.Sprintf("unknown node type %v", n))
	}
}
