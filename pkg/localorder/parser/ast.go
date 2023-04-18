package parser

type Node interface {
	Type() NodeType
	Position() Position
	Children() []Node
}

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
	NodeEquals
	NodeList
	NodeLeftParen
	NodeRightParen
	NodeFunction
	NodeArgument
	NodeLeftBracket
	NodeRightBracket
	NodeListElement
	NodeComma
	NodeSolveFn
	NodeSolveLegacyFn
	NodeRequirementFn
	NodeAppendFn
	NodeString
	NodeComment
)

var literalNodes = map[NodeType]bool{
	NodeString:        true,
	NodeComment:       true,
	NodeLet:           true,
	NodeIn:            true,
	NodeColon:         true,
	NodeEquals:        true,
	NodeLeftParen:     true,
	NodeRightParen:    true,
	NodeLeftBracket:   true,
	NodeRightBracket:  true,
	NodeComma:         true,
	NodeSolveFn:       true,
	NodeSolveLegacyFn: true,
	NodeIdentifier:    true,
}

var nodeNames = map[NodeType]string{
	NodeFile:          "File",
	NodeExpression:    "Expression",
	NodeApplication:   "Application",
	NodeLet:           "Let",
	NodeIn:            "In",
	NodeColon:         "Colon",
	NodeBinding:       "Binding",
	NodeAssignment:    "Assignment",
	NodeIdentifier:    "Identifier",
	NodeEquals:        "Equals",
	NodeList:          "List",
	NodeLeftParen:     "LeftParen",
	NodeRightParen:    "RightParen",
	NodeFunction:      "Function",
	NodeArgument:      "Argument",
	NodeLeftBracket:   "LeftBracket",
	NodeRightBracket:  "RightBracket",
	NodeListElement:   "ListElement",
	NodeComma:         "Comma",
	NodeSolveFn:       "SolveFn",
	NodeSolveLegacyFn: "SolveLegacyFn",
	NodeString:        "String",
	NodeComment:       "Comment",
}

func (t NodeType) String() string {
	return nodeNames[t]
}

func (t NodeType) HasLiteral() bool {
	return literalNodes[t]
}

type NodeElement struct {
	t        NodeType
	pos      Position
	lit      string
	children []*NodeElement
}

func (n *NodeElement) Type() NodeType {
	return n.t
}

func (n *NodeElement) Position() Position {
	return n.pos
}

func (n *NodeElement) Literal() string {
	return n.lit
}

func (n *NodeElement) Children() []*NodeElement {
	return n.children
}

func (n *NodeElement) AddChild(child *NodeElement) {
	n.children = append(n.children, child)
}

type Tree struct {
	Root *NodeElement
}

func (t *Tree) AddChild(child *NodeElement) {
	t.Root.children = append(t.Root.children, child)
}

func (t *Tree) Children() []*NodeElement {
	var result []*NodeElement
	return getChildren(t.Root, result)
}

func (t *Tree) Find(pos Position) *NodeElement {
	return find(t.Root, pos)
}

func find(node *NodeElement, pos Position) *NodeElement {
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

func getChildren(node *NodeElement, result []*NodeElement) []*NodeElement {
	for _, c := range node.children {
		result = append(result, c)
		result = getChildren(c, result)
	}
	return result
}

func (t *Tree) String() string {
	return t.Root.String()
}

type walkFunc func(node *NodeElement) error

func (t *Tree) Walk(fn walkFunc) {
	walk(t.Root, fn)
}

func walk(node *NodeElement, fn walkFunc) {
	fn(node)
	for _, c := range node.children {
		walk(c, fn)
	}
}

func (n *NodeElement) String() string {
	var result string
	result += n.t.String()
	if n.lit != "" {
		result += " " + n.lit
	}
	// result += " " + n.pos.String()
	for _, c := range n.children {
		result += " " + c.String()
	}
	return result
}
