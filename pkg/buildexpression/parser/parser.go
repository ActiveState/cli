package parser

import (
	"github.com/ActiveState/cli/internal/errs"
)

type lexed struct {
	pos Position
	tok Token
	lit string
}

type Parser struct {
	lexer *Lexer
	tree  *Tree

	// The following all represent the current state of the parser
	pos Position
	tok Token
	lit string

	lexed   []lexed
	current int
}

func New(data []byte) (*Parser, error) {
	lexedData := make([]lexed, 0)

	lexer := NewLexer(data)
	var (
		pos Position
		tok Token
		lit string
		err error
	)
	for tok != EOF {
		pos, tok, lit, err = lexer.Scan()
		if err != nil {
			return nil, errs.Wrap(err, "Failed to build internal list of tokens")
		}
		lexedData = append(lexedData, lexed{pos, tok, lit})
	}

	return &Parser{
		lexer: NewLexer(data),
		lexed: lexedData,
	}, nil
}

func (p *Parser) next() {
	if p.current >= len(p.lexed) {
		p.pos = Position{}
		p.tok = EOF
		p.lit = ""
		return
	}

	currentLexed := p.lexed[p.current]
	p.pos = currentLexed.pos
	p.tok = currentLexed.tok
	p.lit = currentLexed.lit
	p.current++
}

func (p *Parser) peek() Token {
	if p.current >= len(p.lexed) {
		return EOF
	}
	return p.lexed[p.current].tok
}

func (p *Parser) newNode(t NodeType) *Node {
	elem := &Node{
		t:   t,
		pos: p.pos,
	}

	if t.HasLiteral() {
		elem.lit = p.lit
	}

	return elem
}

func (p *Parser) expectToken(tok Token, parent *Node, node NodeType) error {
	if p.tok != tok {
		return errs.New("Expected token: %s, got: %s@%d%d", tok.String(), p.tok.String(), p.pos.Line, p.pos.Column)
	}
	parent.AddChild(p.newNode(node))

	p.next()
	return nil
}

func (p *Parser) Parse() (*Tree, error) {
	result := Tree{
		Root: &Node{
			t: NodeFile,
		},
	}
	p.tree = &result

	p.next()

	// This should be the start of the JSON expression
	err := p.expectToken(L_CURL, result.Root, NodeLeftCurlyBracket)
	if err != nil {
		return nil, errs.Wrap(err, "Expect failed")
	}

	for p.tok != EOF {
		switch p.tok {
		case LET:
			err = p.parseExpression(result.Root)
			if err != nil {
				return nil, errs.Wrap(err, "Failed to parse expression")
			}
			if p.tok == R_CURL {
				result.Root.AddChild(p.newNode(NodeRightCurlyBracket))
			}
		case IN:
			err = p.parseIn(result.Root)
			if err != nil {
				return nil, errs.Wrap(err, "Failed to parse in")
			}
			if p.tok == COMMA {
				result.Root.AddChild(p.newNode(NodeComma))
			}
		}

		p.next()
	}

	return &result, nil
}

func (p *Parser) parseExpression(root *Node) error {
	if !isExpression(p.tok) {
		return errs.New("Expected expression")
	}
	expressionNode := p.newNode(NodeExpression)
	root.AddChild(expressionNode)

	expressionNode.AddChild(p.newNode(NodeLet))

	p.next()

	err := p.expectToken(COLON, expressionNode, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	err = p.expectToken(L_CURL, expressionNode, NodeLeftCurlyBracket)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	err = p.parseBinding(expressionNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse binding")
	}

	return nil
}

func (p *Parser) parseBinding(node *Node) error {
	bindingNode := p.newNode(NodeBinding)
	node.AddChild(bindingNode)

	for {
		if !p.isAssignment() {
			break
		}

		err := p.parseAssignment(bindingNode)
		if err != nil {
			return errs.Wrap(err, "Failed to parse assignment")
		}
	}

	return nil
}

func (p *Parser) parseAssignment(node *Node) error {
	if !p.isAssignment() {
		return errs.New("Expected assignment")
	}

	assignmentNode := p.newNode(NodeAssignment)
	node.AddChild(assignmentNode)

	err := p.parseIdentifier(assignmentNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse identifier")
	}

	err = p.expectToken(COLON, assignmentNode, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	// In this case the next token should be a function identifier
	if p.tok == L_CURL && isFunctionIdentifier(p.peek()) {
		assignmentNode.AddChild(p.newNode(NodeLeftCurlyBracket))
		p.next()

		if !isFunctionIdentifier(p.tok) {
			return errs.New("Expected function identifier")
		}

		err = p.parseApplication(assignmentNode)
		if err != nil {
			return errs.Wrap(err, "Failed to parse application")
		}

		err = p.expectToken(R_CURL, assignmentNode, NodeRightCurlyBracket)
		if err != nil {
			return errs.Wrap(err, "Expect failed")
		}

		if p.tok == COMMA {
			assignmentNode.AddChild(p.newNode(NodeComma))
			p.next()
		}
	}

	var msg string
	switch p.tok {
	case STRING:
		err = p.parseString(assignmentNode)
		msg = "Failed to parse string"
	case IDENTIFIER:
		err = p.parseIdentifier(assignmentNode)
		msg = "Failed to parse identifier"
	case IN, R_CURL:
		// If we encounter an IN or R_CURL token then we've reached the end of the binding
		return nil
	default:
		err = p.parseList(assignmentNode)
		msg = "Failed to parse list"
	}
	if err != nil {
		return errs.Wrap(err, msg)
	}

	return p.expectToken(R_CURL, assignmentNode, NodeRightCurlyBracket)
}

func (p *Parser) parseIdentifier(node *Node) error {
	if !isIdentifier(p.tok) {
		return errs.New("Expected identifier, got: %s", p.tok.String())
	}

	identifierNode := p.newNode(NodeIdentifier)
	node.AddChild(identifierNode)

	p.next()
	return nil
}

func (p *Parser) parseApplication(node *Node) error {
	if !isFunctionIdentifier(p.tok) {
		return errs.New("Expected function identifier, got: %s, lit: %s", p.tok.String(), p.lit)
	}

	applicationNode := p.newNode(NodeApplication)
	node.AddChild(applicationNode)

	err := p.parseFunctionIdentifier(applicationNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse function identifier")
	}

	err = p.expectToken(COLON, applicationNode, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	err = p.expectToken(L_CURL, applicationNode, NodeLeftCurlyBracket)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	err = p.parseArguments(applicationNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse arguments")
	}

	return p.expectToken(R_CURL, applicationNode, NodeRightCurlyBracket)
}

func (p *Parser) parseFunctionIdentifier(node *Node) error {
	if p.tok != F_SOLVE && p.tok != F_SOLVELEGACY && p.tok != F_MERGE {
		return errs.New("Unknown function identifier")
	}

	functionIdentifierNode := p.newNode(NodeFunction)
	node.AddChild(functionIdentifierNode)

	switch p.tok {
	case F_SOLVE:
		functionIdentifierNode.AddChild(p.newNode(NodeSolveFn))
	case F_SOLVELEGACY:
		functionIdentifierNode.AddChild(p.newNode(NodeSolveLegacyFn))
	case F_MERGE:
		functionIdentifierNode.AddChild(p.newNode(NodeMergeFn))
	}

	p.next()
	return nil
}

func (p *Parser) parseArguments(node *Node) error {
	for {
		if !isIdentifier(p.tok) {
			break
		}

		err := p.parseIdentifier(node)
		if err != nil {
			return errs.Wrap(err, "Failed to parse identifier")
		}

		err = p.expectToken(COLON, node, NodeColon)
		if err != nil {
			return errs.Wrap(err, "Expect failed")
		}

		var msg string
		switch p.tok {
		case STRING:
			err = p.parseString(node)
			msg = "Failed to parse string"
		case IDENTIFIER:
			err = p.parseIdentifier(node)
			msg = "Failed to parse identifier"
		default:
			err = p.parseList(node)
			msg = "Failed to parse list from arguments"
		}
		if err != nil {
			return errs.Wrap(err, msg)
		}

		if p.tok == COMMA {
			node.AddChild(p.newNode(NodeComma))
			p.next()
			continue
		}
	}

	return nil
}

func (p *Parser) parseList(node *Node) error {
	if p.tok != L_BRACKET {
		return errs.New("Expected left bracket, got: %s@pos:%d:%d", p.lit, p.pos.Column, p.pos.Line)
	}

	listNode := &Node{
		t:   NodeList,
		pos: p.pos,
	}
	node.AddChild(listNode)
	listNode.AddChild(p.newNode(NodeLeftBracket))

	p.next()

	err := p.parseListElements(listNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse list element")
	}

	return p.expectToken(R_BRACKET, listNode, NodeRightBracket)
}

func (p *Parser) parseListElements(node *Node) error {
	for {
		if p.tok != L_CURL && !isString(p.tok) && p.tok != COMMA {
			break
		}

		if p.tok == COMMA {
			node.AddChild(p.newNode(NodeComma))

			p.next()
			continue
		}

		var err error
		var msg string
		switch p.tok {
		case L_CURL:
			err = p.parseListObject(node)
			msg = "Failed to parse list object"
		case STRING:
			err = p.parseListString(node)
			msg = "Failed to parse list string"
		}
		if err != nil {
			return errs.Wrap(err, msg)
		}
	}

	return nil
}

func (p *Parser) parseListString(node *Node) error {
	elementNode := p.newNode(NodeListElement)
	node.AddChild(elementNode)

	return p.parseString(elementNode)
}

func (p *Parser) parseListObject(node *Node) error {
	elementNode := p.newNode(NodeListElement)
	node.AddChild(elementNode)

	objectNode := p.newNode(NodeObject)
	elementNode.AddChild(objectNode)

	err := p.expectToken(L_CURL, objectNode, NodeLeftCurlyBracket)
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	for {
		if p.tok != IDENTIFIER {
			break
		}

		err := p.parseObjectAttribute(objectNode)
		if err != nil {
			return errs.Wrap(err, "Failed to parse object attribute")
		}

		if p.tok == COMMA {
			objectNode.AddChild(p.newNode(NodeComma))
			p.next()
		}
	}

	return p.expectToken(R_CURL, objectNode, NodeRightCurlyBracket)
}

func (p *Parser) parseObjectAttribute(node *Node) error {
	if p.tok != IDENTIFIER {
		return errs.New("Expected identifier")
	}

	err := p.parseIdentifier(node)
	if err != nil {
		return errs.Wrap(err, "Failed to parse identifier")
	}

	err = p.expectToken(COLON, node, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	var msg string
	switch p.tok {
	case STRING:
		err = p.parseString(node)
		msg = "Failed to parse string"
	case L_CURL:
		err = p.parseListObject(node)
		msg = "Failed to parse list object"
	default:
		err = p.parseList(node)
		msg = "Failed to parse list"
	}
	if err != nil {
		return errs.Wrap(err, msg)
	}

	return nil
}

func (p *Parser) parseString(node *Node) error {
	if p.tok != STRING {
		return errs.New("Expected string")
	}
	node.AddChild(p.newNode(NodeString))

	p.next()
	return nil
}

func (p *Parser) parseIn(node *Node) error {
	if p.tok != IN {
		return errs.New("Expected in")
	}
	node.AddChild(p.newNode(NodeIn))

	p.next()

	err := p.expectToken(COLON, node, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	var msg string
	switch p.tok {
	case STRING:
		err = p.parseString(node)
		msg = "Failed to parse string"
	case IDENTIFIER:
		err = p.parseApplication(node)
		msg = "Failed to parse application"
	default:
		return errs.New("Expected string or identifier, got: %s", p.lit)
	}
	if err != nil {
		return errs.Wrap(err, msg)
	}

	return nil
}

func (p *Parser) isAssignment() bool {
	return p.tok == IDENTIFIER && p.peek() == COLON && p.tok != IN
}

func isExpression(tok Token) bool {
	// Currently only supporting Let, can be expanded to
	// support application and identifier later
	return tok == LET
}

func isIdentifier(tok Token) bool {
	return tok == IDENTIFIER
}

func isFunctionIdentifier(tok Token) bool {
	return tok == F_SOLVE || tok == F_SOLVELEGACY || tok == F_MERGE
}

func isString(tok Token) bool {
	return tok == STRING
}
