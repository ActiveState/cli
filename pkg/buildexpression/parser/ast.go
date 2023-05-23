package parser

type NodeType int

const (
	NodeFile NodeType = iota
	NodeExpression
	NodeApplication
	NodeLet
	NodeIn
	NodeColon
	NodeBinding
	NodeAssignment
	NodeIdentifier
	NodeList
	NodeFunction
	NodeArgument
	NodeLeftBracket
	NodeRightBracket
	NodeLeftCurlyBracket
	NodeRightCurlyBracket
	NodeListElement
	NodeComma
	NodeSolveFn
	NodeSolveLegacyFn
	NodeMergeFn
	NodeString
	NodeObject
)

var literalNodes = map[NodeType]bool{
	NodeString:            true,
	NodeLet:               true,
	NodeIn:                true,
	NodeColon:             true,
	NodeLeftBracket:       true,
	NodeRightBracket:      true,
	NodeLeftCurlyBracket:  true,
	NodeRightCurlyBracket: true,
	NodeComma:             true,
	NodeSolveFn:           true,
	NodeSolveLegacyFn:     true,
	NodeMergeFn:           true,
	NodeIdentifier:        true,
}

var nodeNames = map[NodeType]string{
	NodeFile:              "File",
	NodeExpression:        "Expression",
	NodeApplication:       "Application",
	NodeLet:               "Let",
	NodeIn:                "In",
	NodeColon:             "Colon",
	NodeBinding:           "Binding",
	NodeAssignment:        "Assignment",
	NodeIdentifier:        "Identifier",
	NodeList:              "List",
	NodeFunction:          "Function",
	NodeArgument:          "Argument",
	NodeLeftBracket:       "LeftBracket",
	NodeRightBracket:      "RightBracket",
	NodeLeftCurlyBracket:  "LeftCurlyBracket",
	NodeRightCurlyBracket: "RightCurlyBracket",
	NodeListElement:       "ListElement",
	NodeComma:             "Comma",
	NodeSolveFn:           "SolveFn",
	NodeSolveLegacyFn:     "SolveLegacyFn",
	NodeMergeFn:           "MergeFn",
	NodeString:            "String",
	NodeObject:            "Object",
}

func (t NodeType) String() string {
	return nodeNames[t]
}

func (t NodeType) HasLiteral() bool {
	return literalNodes[t]
}

type Node struct {
	t        NodeType
	pos      Position
	lit      string
	children []*Node
}

func (n *Node) Type() NodeType {
	return n.t
}

func (n *Node) Position() Position {
	return n.pos
}

func (n *Node) Literal() string {
	return n.lit
}

func (n *Node) Children() []*Node {
	return n.children
}

func (n *Node) AddChild(child *Node) {
	n.children = append(n.children, child)
}

type Tree struct {
	Root *Node
}

func (t *Tree) AddChild(child *Node) {
	t.Root.children = append(t.Root.children, child)
}

func (t *Tree) Children() []*Node {
	var result []*Node
	return getChildren(t.Root, result)
}

func (t *Tree) Find(pos Position) *Node {
	return find(t.Root, pos)
}

func find(node *Node, pos Position) *Node {
	if node.pos == pos {
		return node
	}

	for _, c := range node.children {
		found := find(c, pos)
		if found != nil {
			return found
		}
	}

	return nil
}

func getChildren(node *Node, result []*Node) []*Node {
	for _, c := range node.children {
		result = append(result, c)
		result = getChildren(c, result)
	}
	return result
}

func (t *Tree) String() string {
	return t.Root.String()
}

type walkFunc func(node *Node) error

func (t *Tree) Walk(fn walkFunc) {
	walk(t.Root, fn)
}

func walk(node *Node, fn walkFunc) {
	fn(node)
	for _, c := range node.children {
		walk(c, fn)
	}
}

func (n *Node) String() string {
	var result string
	result += n.t.String()
	if n.lit != "" {
		result += " " + n.lit
	}
	for _, c := range n.children {
		result += " " + c.String()
	}
	return result
}
