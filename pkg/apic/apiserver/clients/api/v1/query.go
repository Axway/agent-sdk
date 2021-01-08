package v1

import (
	"fmt"
	"sort"
	"strings"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// QueryStringer helps print a query
type QueryStringer struct {
	QueryNode
}

func (qs QueryStringer) String() string {
	v := rsqlVisitor{strings.Builder{}}
	v.Visit(qs)

	return v.String()
}

// Visitor visits a QueryNode
type Visitor interface {
	Visit(QueryNode)
}

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

type namesNode []string

func (n namesNode) Accept(v Visitor) {
	v.Visit(n)
}

func (n *attrNode) Accept(v Visitor) {
	v.Visit(n)
}

func (n tagNode) Accept(v Visitor) {
	v.Visit(n)
}

type referenceNode struct {
	gvk  apiv1.GroupVersionKind
	name string
}

func (r *referenceNode) Accept(v Visitor) {
	v.Visit(r)
}

// AttrIn creates a query that matches resources with attribute key and  any of values
func AttrIn(key string, values ...string) QueryNode {
	return &attrNode{
		key,
		values,
	}
}

// TagsIn creates a query that matches resources with any of the tag values
func TagsIn(values ...string) QueryNode {
	return tagNode(values)
}

func AllTags(values ...string) QueryNode {
	subNodes := make([]QueryNode, len(values))
	for i, value := range values {
		subNodes[i] = TagsIn(value)
	}

	return andNode(subNodes)
}

// AnyAttr creates a query that matches resources with any of the attributes
func AnyAttr(attrs map[string]string) QueryNode {
	nodes := make([]QueryNode, len(attrs))
	i := 0

	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		nodes[i] = AttrIn(key, attrs[key])
		i++
	}
	return orNode(nodes)
}

// AllAttr creates a query that matches resources with all of the attributes
func AllAttr(attrs map[string]string) QueryNode {
	nodes := make([]QueryNode, len(attrs))
	i := 0

	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		nodes[i] = AttrIn(key, attrs[key])
		i++
	}
	return andNode(nodes)
}

// Names creates a query that matches any of the passed names
func Names(names ...string) QueryNode {
	return namesNode(names)
}

// Or creates a query that ors two or more subqueries
func Or(first, second QueryNode, rest ...QueryNode) QueryNode {
	nodes := make([]QueryNode, len(rest)+2)
	nodes[0] = first
	nodes[1] = second

	return orNode(append(nodes, rest...))
}

// And creates a query that ands two or more subqueries
func And(first, second QueryNode, rest ...QueryNode) QueryNode {
	nodes := make([]QueryNode, len(rest)+2)
	nodes[0] = first
	nodes[1] = second

	return andNode(append(nodes, rest...))
}

// Reference create a query by reference to resource kind and name
func Reference(gvk apiv1.GroupVersionKind, name string) QueryNode {
	return &referenceNode{
		gvk:  gvk,
		name: name,
	}
}

// rsqlVisitor builds an RSQL string by visiting QueryNodes
type rsqlVisitor struct {
	b strings.Builder
}

func newRSQLVisitor() *rsqlVisitor {
	return &rsqlVisitor{
		b: strings.Builder{},
	}
}

func (rv *rsqlVisitor) String() string {
	return rv.b.String()
}

func (rv *rsqlVisitor) Visit(node QueryNode) {

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
	case namesNode:
		switch len(n) {
		case 0:
			rv.b.WriteString(`name==""`)
		case 1:
			rv.b.WriteString(fmt.Sprintf(`name=="%s"`, n[0]))
		default:
			rv.b.WriteString(fmt.Sprintf(`name=in=("%s")`, strings.Join(n, `","`)))
		}
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
	case *referenceNode:
		rv.b.WriteString(fmt.Sprintf(`metadata.references.name==%s;metadata.references.kind==%s`, n.name, n.gvk.GroupKind.Kind))
	default:
		panic(fmt.Sprintf("unknown node type %v", n))
	}
}
