package publiccode

import (
	"github.com/goccy/go-yaml/ast"
)

// maxAliasNodes is how many nodes the aliases of a document may add to it
// once expanded. The decoder copies the anchored value at each alias, so
// anchors that alias each other grow a small document exponentially.
// Manifests that reuse a few values stay far below this.
const maxAliasNodes = 10_000

// anchorIndex maps anchor names to the anchors with that name, in source
// order, including those under keys a decode never reaches.
type anchorIndex map[string][]*ast.AnchorNode

func indexAnchors(node ast.Node) anchorIndex {
	anchors := anchorIndex{}
	ast.Walk(anchors, node)

	return anchors
}

//nolint:ireturn // ast.Walk needs a Visitor back.
func (a anchorIndex) Visit(node ast.Node) ast.Visitor {
	if anchor, ok := node.(*ast.AnchorNode); ok {
		name := anchor.Name.GetToken().Value
		a[name] = append(a[name], anchor)
	}

	return a
}

// resolve returns the last anchor named like the alias that precedes it,
// as YAML says an alias refers to the most recent anchor, or nil.
func (a anchorIndex) resolve(alias *ast.AliasNode) *ast.AnchorNode {
	tok := alias.Value.GetToken()

	var target *ast.AnchorNode

	for _, anchor := range a[tok.Value] {
		if anchor.GetToken().Position.Offset >= tok.Position.Offset {
			break
		}

		target = anchor
	}

	return target
}

// aliasExpansion counts the nodes a document has once its aliases are
// expanded, in time linear in the size of the document.
type aliasExpansion struct {
	anchors anchorIndex
	sizes   map[*ast.AnchorNode]int
	inside  map[*ast.AnchorNode]bool
	added   int
	over    *ast.AliasNode
}

// findAliasOverflow returns the alias that takes the nodes added by alias
// expansion past maxAliasNodes, in source order, or nil.
func findAliasOverflow(node ast.Node) *ast.AliasNode {
	e := aliasExpansion{
		anchors: indexAnchors(node),
		sizes:   map[*ast.AnchorNode]int{},
		inside:  map[*ast.AnchorNode]bool{},
	}
	ast.Walk(&e, node)

	return e.over
}

//nolint:ireturn // ast.Walk needs a Visitor back.
func (e *aliasExpansion) Visit(node ast.Node) ast.Visitor {
	if e.over != nil {
		return nil
	}

	alias, ok := node.(*ast.AliasNode)
	if !ok {
		return e
	}

	e.added = saturatingAdd(e.added, e.size(alias))
	if e.added > maxAliasNodes {
		e.over = alias
	}

	return nil
}

// size returns the number of nodes under node with aliases expanded,
// capped just past maxAliasNodes.
func (e *aliasExpansion) size(node ast.Node) int {
	switch n := node.(type) {
	case nil:
		return 0
	case *ast.DocumentNode:
		return e.size(n.Body)
	case *ast.TagNode:
		return e.size(n.Value)
	case *ast.AnchorNode:
		return e.anchorSize(n)
	case *ast.AliasNode:
		anchor := e.anchors.resolve(n)
		if anchor == nil {
			return 1
		}

		return e.anchorSize(anchor)
	case *ast.MappingKeyNode:
		return e.size(n.Value)
	case *ast.MappingValueNode:
		return saturatingAdd(e.size(n.Key), e.size(n.Value))
	case *ast.MappingNode:
		total := 1
		for _, value := range n.Values {
			total = saturatingAdd(total, e.size(value))
		}

		return total
	case *ast.SequenceNode:
		total := 1
		for _, value := range n.Values {
			total = saturatingAdd(total, e.size(value))
		}

		return total
	default:
		return 1
	}
}

func (e *aliasExpansion) anchorSize(anchor *ast.AnchorNode) int {
	if size, ok := e.sizes[anchor]; ok {
		return size
	}

	// An anchor can hold an alias of itself, "&a [*a]", which the decoder
	// doesn't expand.
	if e.inside[anchor] {
		return 1
	}

	e.inside[anchor] = true
	size := e.size(anchor.Value)
	delete(e.inside, anchor)

	e.sizes[anchor] = size

	return size
}

func saturatingAdd(a, b int) int {
	return min(a+b, maxAliasNodes+1)
}
