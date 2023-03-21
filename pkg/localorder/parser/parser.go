package parser

import (
	"github.com/ActiveState/cli/internal/errs"
)

type Parser struct {
	lexer *Lexer
	tree  *Tree

	// The following all represent the current state of the parser
	pos Position
	tok Token
	lit string
}

func New(data []byte) *Parser {
	return &Parser{
		lexer: NewLexer(data),
	}
}

func (p *Parser) Next() error {
	parentPos := p.pos
	err := p.next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.tok == COMMENT {
		parentNode := p.tree.Find(parentPos)
		if parentNode == nil {
			return errs.New("Failed to find parent node for comment")
		}

		parentNode.AddChild(&NodeElement{
			t:   NodeComment,
			pos: p.pos,
			lit: p.lit,
		})

		err = p.next()
		if err != nil {
			return errs.Wrap(err, "Failed to scan")
		}
	}

	return nil
}

func (p *Parser) next() error {
	pos, tok, lit, err := p.lexer.Scan()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}
	p.pos = pos
	p.tok = tok
	p.lit = lit
	return nil
}

func (p *Parser) newNode(t NodeType) *NodeElement {
	elem := &NodeElement{
		t:   t,
		pos: p.pos,
	}

	if t.HasLiteral() {
		elem.lit = p.lit
	}

	return elem
}

func (p *Parser) Parse() (*Tree, error) {
	result := Tree{
		Root: &NodeElement{
			t: NodeFile,
		},
	}
	p.tree = &result

	// TODO: May want a method on the lexer that returns all of the tokens
	err := p.Next()
	if err != nil {
		return nil, errs.Wrap(err, "Failed to scan")
	}

	// TODO: Rather than making the parse functions responsible for calling Next
	// we could have a for loop here that calls Next and then calls the appropriate
	// parse function.
	err = p.ParseExpression(result.Root)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to parse expression")
	}

	return &result, nil
}

func (p *Parser) ParseExpression(root *NodeElement) error {
	// Right now this is just parsing a let statement
	if !p.IsExpression() {
		return errs.New("Expected expression")
	}
	expressionNode := p.newNode(NodeLet)
	root.AddChild(expressionNode)

	err := p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.tok != COLON {
		return errs.New("Expected colon")
	}
	expressionNode.AddChild(p.newNode(NodeColon))

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	err = p.ParseBinding(expressionNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse binding")
	}

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.tok != IN {
		return errs.New("Expected in, got: %s", p.lit)
	}
	expressionNode.AddChild(p.newNode(NodeIn))

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.tok != COLON {
		return errs.New("Expected colon")
	}
	expressionNode.AddChild(p.newNode(NodeColon))

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.IsIdentifier() {
		return p.ParseIdentifier(expressionNode)
	}

	return p.ParseExpression(root)
}

func (p *Parser) ParseBinding(node *NodeElement) error {
	if !p.IsBinding() {
		return errs.New("Expected binding")
	}

	bindingNode := p.newNode(NodeBinding)
	node.AddChild(bindingNode)

	err := p.ParseAssignment(bindingNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse assignment")
	}

	return nil
}

func (p *Parser) ParseAssignment(node *NodeElement) error {
	assignmentNode := p.newNode(NodeAssignment)
	node.AddChild(assignmentNode)

	err := p.ParseIdentifier(assignmentNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse identifier")
	}

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.tok != EQUALS {
		return errs.New("Expected equals")
	}
	assignmentNode.AddChild(p.newNode(NodeEquals))

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.IsFunctionIdentifier() {
		return p.ParseApplication(assignmentNode)
	}

	return p.ParseList(assignmentNode)
}

func (p *Parser) ParseIdentifier(node *NodeElement) error {
	if !p.IsIdentifier() {
		return errs.New("Expected identifier")
	}

	identifierNode := p.newNode(NodeIdentifier)
	node.AddChild(identifierNode)

	return nil
}

func (p *Parser) ParseApplication(node *NodeElement) error {
	if !p.IsFunctionIdentifier() {
		return errs.New("Expected function identifier")
	}

	applicationNode := p.newNode(NodeApplication)
	node.AddChild(applicationNode)

	err := p.ParseFunctionIdentifier(applicationNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse function identifier")
	}

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.tok != L_PAREN {
		return errs.New("Expected left parenthesis")
	}
	applicationNode.AddChild(p.newNode(NodeLeftParen))

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	err = p.ParseArguments(applicationNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse arguments")
	}

	if p.tok != R_PAREN {
		return errs.New("Expected right parenthesis, got %s", p.lit)
	}

	applicationNode.AddChild(p.newNode(NodeRightParen))

	return nil
}

func (p *Parser) ParseFunctionIdentifier(node *NodeElement) error {
	if p.tok != F_SOLVE && p.tok != F_SOLVELEGACY {
		return errs.New("Unknown function identifier")
	}

	functionIdentifierNode := p.newNode(NodeFunction)
	node.AddChild(functionIdentifierNode)

	if p.tok == F_SOLVE {
		functionIdentifierNode.AddChild(p.newNode(NodeSolveFn))
	} else if p.tok == F_SOLVELEGACY {
		functionIdentifierNode.AddChild(p.newNode(NodeSolveLegacyFn))
	}

	return nil
}

func (p *Parser) ParseArguments(node *NodeElement) error {
	for p.IsBinding() {
		err := p.ParseBinding(node)
		if err != nil {
			return errs.Wrap(err, "Failed to parse binding")
		}

		if p.tok == R_PAREN {
			break
		}

		err = p.Next()
		if err != nil {
			return errs.Wrap(err, "Failed to scan")
		}
	}

	return nil
}

func (p *Parser) ParseList(node *NodeElement) error {
	if p.tok != L_BRACKET {
		return errs.New("Expected left bracket")
	}

	listNode := &NodeElement{
		t:   NodeList,
		pos: p.pos,
	}
	node.AddChild(listNode)
	listNode.AddChild(p.newNode(NodeLeftBracket))

	err := p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	err = p.ParseListElements(listNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse list element")
	}

	// err = p.Next()
	// if err != nil {
	// 	return errs.Wrap(err, "Failed to scan")
	// }

	if p.tok != R_BRACKET {
		return errs.New("Expected right bracket, current lit: %s", p.lit)
	}
	listNode.AddChild(p.newNode(NodeRightBracket))

	return nil
}

func (p *Parser) ParseListElements(node *NodeElement) error {
	// TODO: Can likely improve the conditions of this loop
	for p.tok == STRING || p.tok == COMMA {
		var err error
		if p.tok == COMMA {
			node.AddChild(p.newNode(NodeComma))

			err = p.Next()
			if err != nil {
				return errs.Wrap(err, "Failed to scan")
			}
		}

		err = p.ParseListElement(node)
		if err != nil {
			return errs.Wrap(err, "Failed to parse list element")
		}

		err = p.Next()
		if err != nil {
			return errs.Wrap(err, "Failed to scan")
		}

		if p.tok == R_BRACKET {
			break
		}
	}

	return nil
}

func (p *Parser) ParseListElement(node *NodeElement) error {
	elementNode := p.newNode(NodeListElement)
	node.AddChild(elementNode)
	return p.ParseString(elementNode)
}

func (p *Parser) ParseString(node *NodeElement) error {
	if p.tok != STRING {
		return errs.New("Expected string")
	}
	node.AddChild(p.newNode(NodeString))

	// TODO: Should increment here?
	err := p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	return nil
}

func (p *Parser) IsExpression() bool {
	return p.tok == LET
}

func (p *Parser) IsBinding() bool {
	return p.tok == IDENTIFIER
}

func (p *Parser) IsIdentifier() bool {
	return p.tok == IDENTIFIER
}

func (p *Parser) IsFunctionIdentifier() bool {
	return p.tok == F_SOLVE || p.tok == F_SOLVELEGACY
}
