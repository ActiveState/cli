package yamlcomment

import (
	"reflect"
	"sort"
	"strings"

	"github.com/maruel/natural"
	yaml "gopkg.in/yaml.v3"
)

const (
	HeadCommentTag = "hc"
	LineCommentTag = "lc"
	FootCommentTag = "fc"
	OmitemptyTag   = "omitempty"
	InlineTag      = "inline"
	FlowTag        = "flow"
)

type option struct {
	fieldName string
	omitempty bool
	skip      bool
	inline    bool
	flow      bool
}

type comment struct {
	HeadComment string
	LineComment string
	FootComment string
}

func setComment(cm *comment, tag reflect.StructTag) {
	if cm == nil {
		return
	}
	cm.HeadComment = tag.Get(HeadCommentTag)
	cm.LineComment = tag.Get(LineCommentTag)
	cm.FootComment = tag.Get(FootCommentTag)
}

func newComment(tag reflect.StructTag) *comment {
	cm := new(comment)
	setComment(cm, tag)
	return cm
}

type CommentEncoder struct {
	encoder *yaml.Encoder
}

func NewEncoder(encoder *yaml.Encoder) *CommentEncoder {
	return &CommentEncoder{
		encoder: encoder,
	}
}

func (e *CommentEncoder) Encode(v any) error {
	node, err := anyToYamlNode(v, false)
	if err != nil {
		return err
	}
	return e.encoder.Encode(node)
}

func Marshal(v any) ([]byte, error) {
	node, err := anyToYamlNode(v, false)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(node)
}

func isZero(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	return value.IsZero()
}

func isNil(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func parseTags(tag reflect.StructTag) (*option, *comment) {
	yamlTags := strings.Split(tag.Get("yaml"), ",")

	var op = &option{
		fieldName: yamlTags[0],
	}

	if op.fieldName == "-" {
		op.skip = true
		return op, nil
	}

	for _, part := range yamlTags[1:] {
		switch part {
		case OmitemptyTag:
			op.omitempty = true
		case InlineTag:
			op.inline = true
		case FlowTag:
			op.flow = true
		}
	}

	return op, newComment(tag)
}

func AnyToYamlNode(model any) (*yaml.Node, error) {
	return anyToYamlNode(model, false)
}

func anyToYamlNode(model any, skip bool) (*yaml.Node, error) {
	if n, ok := model.(*yaml.Node); ok {
		return n, nil
	}

	if m, ok := model.(yaml.Marshaler); ok && !isNil(reflect.ValueOf(model)) {
		res, err := m.MarshalYAML()
		if err != nil {
			return nil, err
		}

		if n, ok := res.(*yaml.Node); ok {
			return n, nil
		}

		model = res
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	node := new(yaml.Node)

	switch v.Kind() {
	case reflect.Struct:
		node.Kind = yaml.MappingNode

		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanInterface() {
				continue
			}

			op, cm := parseTags(t.Field(i).Tag)

			if op.skip || (op.omitempty && isZero(field)) {
				continue
			}

			if op.fieldName == "" {
				op.fieldName = strings.ToLower(t.Field(i).Name)
			}

			var value any
			if field.CanInterface() {
				value = field.Interface()
			}

			var style yaml.Style
			if op.flow {
				style |= yaml.FlowStyle
			}

			if op.inline {
				child, err := anyToYamlNode(value, skip)
				if err != nil {
					return nil, err
				}

				if child.Kind == yaml.MappingNode || child.Kind == yaml.SequenceNode {
					appendNodes(node, child.Content...)
				}
			} else if err := addToMap(node, op.fieldName, value, cm, style, skip); err != nil {
				return nil, err
			}
		}
	case reflect.Map:
		node.Kind = yaml.MappingNode
		keys := v.MapKeys()
		sort.SliceStable(keys, func(i, j int) bool {
			return natural.Less(keys[i].String(), keys[j].String())
		})

		for i, k := range keys {
			if i != 0 {
				skip = true
			}
			if err := addToMap(node, k.Interface(), v.MapIndex(k).Interface(), nil, 0, skip); err != nil {
				return nil, err
			}
		}
	case reflect.Slice:
		node.Kind = yaml.SequenceNode
		nodes := make([]*yaml.Node, v.Len())

		for i := 0; i < v.Len(); i++ {
			if i != 0 {
				skip = true
			}
			element := v.Index(i)

			var err error

			nodes[i], err = anyToYamlNode(element.Interface(), skip)
			if err != nil {
				return nil, err
			}
		}
		appendNodes(node, nodes...)
	default:
		if err := node.Encode(model); err != nil {
			return nil, err
		}
	}

	return node, nil
}

func appendNodes(dest *yaml.Node, nodes ...*yaml.Node) {
	if dest.Content == nil {
		dest.Content = nodes
		return
	}

	dest.Content = append(dest.Content, nodes...)
}

func addToMap(dest *yaml.Node, fieldName, in any, cm *comment, style yaml.Style, skip bool) error {
	key, err := anyToYamlNode(fieldName, skip)
	if err != nil {
		return err
	}

	value, err := anyToYamlNode(in, skip)
	if err != nil {
		return err
	}
	value.Style = style

	if !skip {
		addComment(key, cm)
	}
	appendNodes(dest, key, value)

	return nil
}

func addComment(node *yaml.Node, cm *comment) {
	if cm == nil {
		return
	}

	node.HeadComment = cm.HeadComment
	node.LineComment = cm.LineComment
	node.FootComment = cm.FootComment
}
