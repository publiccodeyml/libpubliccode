package publiccode

import (
	"cmp"
	"encoding"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

// unknownKey is a mapping key the decoder has no struct field for.
type unknownKey struct {
	path  string
	name  string
	token *token.Token
}

// unknownKeyFinder walks a YAML document against the Go type it decodes into
// and collects every key without a matching field, in source order.
//
// go-yaml's DisallowUnknownField stops at the first unknown key of the first
// struct that has one, and picks it by ranging over a map, so it can't tell
// a user about every unknown key, nor always about the same one.
type unknownKeyFinder struct {
	anchors anchorIndex
	inside  map[ast.Node]bool
	visited map[visit]bool
	seen    map[*token.Token]bool
	keys    []unknownKey
}

type visit struct {
	node ast.Node
	typ  reflect.Type
}

// unmarshalerTypes are the interfaces go-yaml hands a value to instead of
// decoding it field by field.
var unmarshalerTypes = []reflect.Type{
	reflect.TypeFor[yaml.BytesUnmarshaler](),
	reflect.TypeFor[yaml.BytesUnmarshalerContext](),
	reflect.TypeFor[yaml.InterfaceUnmarshaler](),
	reflect.TypeFor[yaml.InterfaceUnmarshalerContext](),
	reflect.TypeFor[yaml.NodeUnmarshaler](),
	reflect.TypeFor[yaml.NodeUnmarshalerContext](),
	reflect.TypeFor[encoding.TextUnmarshaler](),
}

// nonStringTags are the tags that turn a key into something other than a
// string, which the decoder then can't match against a field name.
var nonStringTags = []token.ReservedTagKeyword{
	token.IntegerTag, token.FloatTag, token.NullTag,
	token.BinaryTag, token.BooleanTag, token.TimestampTag,
}

func findUnknownKeys(node ast.Node, typ reflect.Type) []unknownKey {
	f := unknownKeyFinder{
		anchors: indexAnchors(node),
		inside:  map[ast.Node]bool{},
		visited: map[visit]bool{},
		seen:    map[*token.Token]bool{},
	}
	f.walk(node, typ, "")

	// Keys brought in by merge keys come up where they are merged.
	slices.SortStableFunc(f.keys, func(a, b unknownKey) int {
		return cmp.Compare(a.token.Position.Offset, b.token.Position.Offset)
	})

	return f.keys
}

// hasCustomUnmarshaler reports whether go-yaml hands values of typ to the
// type itself, which then decides on its own which keys it accepts.
func hasCustomUnmarshaler(typ reflect.Type) bool {
	ptr := reflect.PointerTo(typ)

	return slices.ContainsFunc(unmarshalerTypes, ptr.Implements)
}

func (f *unknownKeyFinder) walk(node ast.Node, typ reflect.Type, path string) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	switch n := node.(type) {
	case nil:
		return
	case *ast.DocumentNode:
		f.walk(n.Body, typ, path)

		return
	case *ast.TagNode:
		f.walk(n.Value, typ, path)

		return
	case *ast.AnchorNode:
		f.walk(n.Value, typ, path)

		return
	case *ast.AliasNode:
		if anchor := f.anchors.resolve(n); anchor != nil {
			f.walk(anchor.Value, typ, path)
		}

		return
	}

	// Each node is checked once per type: aliases can make the same
	// mapping reachable many times over.
	v := visit{node: node, typ: typ}
	if f.visited[v] {
		return
	}

	f.visited[v] = true

	if hasCustomUnmarshaler(typ) {
		return
	}

	switch typ.Kind() { //nolint:exhaustive // other kinds hold no mapping keys
	case reflect.Struct:
		f.walkStruct(node, typ, path)
	case reflect.Map:
		f.walkMap(node, typ.Elem(), path)
	case reflect.Slice, reflect.Array:
		if seq, ok := node.(*ast.SequenceNode); ok {
			for i, entry := range seq.Values {
				f.walk(entry, typ.Elem(), fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
}

func (f *unknownKeyFinder) walkMap(node ast.Node, elem reflect.Type, path string) {
	f.eachKey(node, func(name string, _ *token.Token, value ast.Node) {
		f.walk(value, elem, joinKey(path, name))
	})
}

func (f *unknownKeyFinder) walkStruct(node ast.Node, typ reflect.Type, path string) {
	fields := structFields(typ)

	f.eachKey(node, func(name string, tok *token.Token, value ast.Node) {
		field, ok := fields[name]
		if !ok {
			// Report a key once, however many aliases lead to it.
			if tok != nil && !f.seen[tok] {
				f.seen[tok] = true
				f.keys = append(f.keys, unknownKey{path: joinKey(path, name), name: name, token: tok})
			}

			return
		}

		f.walk(value, field, joinKey(path, name))
	})
}

// eachKey calls fn for every key of a mapping, following merge keys
// into the mappings they merge, like the decoder does.
func (f *unknownKeyFinder) eachKey(node ast.Node, fn func(name string, tok *token.Token, value ast.Node)) {
	// A node can merge an alias of itself: "&a {<<: *a}" or "&a [*a]".
	if node == nil || f.inside[node] {
		return
	}

	f.inside[node] = true
	defer delete(f.inside, node)

	switch n := node.(type) {
	case *ast.TagNode:
		f.eachKey(n.Value, fn)
	case *ast.AnchorNode:
		f.eachKey(n.Value, fn)
	case *ast.AliasNode:
		if anchor := f.anchors.resolve(n); anchor != nil {
			f.eachKey(anchor.Value, fn)
		}
	case *ast.SequenceNode:
		// Only reached through a merge key: "<<: [*a, *b]".
		for _, entry := range n.Values {
			f.eachKey(entry, fn)
		}
	case ast.MapNode:
		iter := n.MapRange()
		for iter.Next() {
			key := iter.Key()
			if key.IsMergeKey() {
				f.eachKey(iter.Value(), fn)

				continue
			}

			// The decoder only matches string keys against field names.
			name, ok := f.keyName(key)
			if !ok {
				continue
			}

			fn(name, key.GetToken(), iter.Value())
		}
	}
}

// keyName returns the string the decoder turns a mapping key into, looking
// through tags, anchors and aliases, or false if the key is not a string.
func (f *unknownKeyFinder) keyName(key ast.Node) (string, bool) {
	switch n := key.(type) {
	case *ast.StringNode:
		return n.Value, true
	case *ast.AnchorNode:
		return f.keyName(n.Value)
	case *ast.AliasNode:
		if anchor := f.anchors.resolve(n); anchor != nil {
			return f.keyName(anchor.Value)
		}
	case *ast.TagNode:
		tag := token.ReservedTagKeyword(n.Start.Value)
		if n.Directive == nil && slices.Contains(nonStringTags, tag) {
			return "", false
		}

		if n.Directive == nil && tag != token.StringTag {
			return f.keyName(n.Value)
		}

		// "!!str 1" is the string "1".
		if name, ok := f.keyName(n.Value); ok {
			return name, true
		}

		if scalar, ok := n.Value.(ast.ScalarNode); ok && scalar.GetValue() != nil {
			return fmt.Sprint(scalar.GetValue()), true
		}
	}

	return "", false
}

// structFields maps the YAML key of every field of typ, including the
// fields of inline structs, to its type, following go-yaml's naming rules.
func structFields(typ reflect.Type) map[string]reflect.Type {
	fields := map[string]reflect.Type{}

	for field := range typ.Fields() {
		tag := field.Tag.Get("yaml")
		if tag == "" {
			tag = field.Tag.Get("json")
		}

		if tag == "-" || (field.PkgPath != "" && !field.Anonymous) {
			continue
		}

		name, options, _ := strings.Cut(tag, ",")
		if name == "" {
			name = strings.ToLower(field.Name)
		}

		if slices.Contains(strings.Split(options, ","), "inline") {
			inline := field.Type
			for inline.Kind() == reflect.Pointer {
				inline = inline.Elem()
			}

			if inline.Kind() == reflect.Struct {
				maps.Copy(fields, structFields(inline))
			}

			continue
		}

		fields[name] = field.Type
	}

	return fields
}

func joinKey(path, name string) string {
	if path == "" {
		return name
	}

	return path + "." + name
}
