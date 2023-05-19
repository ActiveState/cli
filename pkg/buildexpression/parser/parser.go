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

	toks    []Token
	current int
}

func New(data []byte) *Parser {
	toks := make([]Token, 0)

	l := NewLexer(data)
	var tok Token
	for tok != EOF {
		_, tok, _, _ = l.Scan()
		toks = append(toks, tok)
	}

	return &Parser{
		lexer: NewLexer(data),
		toks:  toks,
	}
}

func (p *Parser) Next() error {
	err := p.next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
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
	p.current++
	return nil
}

func (p *Parser) peek() Token {
	if p.current >= len(p.toks) {
		return EOF
	}
	return p.toks[p.current]
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

	return p.Next()
}

func (p *Parser) Parse() (*Tree, error) {
	result := Tree{
		Root: &Node{
			t: NodeFile,
		},
	}
	p.tree = &result

	// TODO: May want a method on the lexer that returns all of the tokens
	err := p.Next()
	if err != nil {
		return nil, errs.Wrap(err, "Failed to scan")
	}

	// Should be the start of the JSON expression
	err = p.expectToken(L_CURL, result.Root, NodeLeftCurlyBracket)
	if err != nil {
		return nil, errs.Wrap(err, "Expect failed")
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

func (p *Parser) ParseExpression(root *Node) error {
	// Right now this is just parsing a let statement
	if !p.IsExpression() {
		return errs.New("Expected expression")
	}
	expressionNode := p.newNode(NodeExpression)
	root.AddChild(expressionNode)

	expressionNode.AddChild(p.newNode(NodeLet))

	err := p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	err = p.expectToken(COLON, expressionNode, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	err = p.expectToken(L_CURL, expressionNode, NodeLeftCurlyBracket)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	err = p.ParseBinding(expressionNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse binding")
	}

	err = p.expectToken(IN, expressionNode, NodeIn)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	err = p.expectToken(COLON, expressionNode, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	if p.IsFunctionIdentifier() {
		return p.ParseApplication(expressionNode)
	}

	return p.ParseString(expressionNode)
}

func (p *Parser) ParseBinding(node *Node) error {
	bindingNode := p.newNode(NodeBinding)
	node.AddChild(bindingNode)

	for {
		if !p.IsAssignment() {
			break
		}

		err := p.ParseAssignment(bindingNode)
		if err != nil {
			return errs.Wrap(err, "Failed to parse assignment")
		}
	}

	return nil
}

func (p *Parser) IsAssignment() bool {
	// TODO: Not necessarily true
	return p.peek() == COLON && p.tok != IN
}

func (p *Parser) ParseAssignment(node *Node) error {
	if !p.IsAssignment() {
		return errs.New("Expected assignment")
	}

	assignmentNode := p.newNode(NodeAssignment)
	node.AddChild(assignmentNode)

	err := p.ParseIdentifier(assignmentNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse identifier")
	}

	err = p.expectToken(COLON, assignmentNode, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	// In this case the next token should be a function identifier
	// TODO: clean up this conditional
	if p.tok == L_CURL && p.peek() == F_SOLVELEGACY {
		assignmentNode.AddChild(p.newNode(NodeLeftCurlyBracket))
		err = p.Next()
		if err != nil {
			return errs.Wrap(err, "Failed to scan")
		}

		if !p.IsFunctionIdentifier() {
			return errs.New("Expected function identifier")
		}

		err = p.ParseApplication(assignmentNode)
		if err != nil {
			return errs.Wrap(err, "Failed to parse application")
		}

		err = p.expectToken(R_CURL, assignmentNode, NodeRightCurlyBracket)
		if err != nil {
			return errs.Wrap(err, "Expect failed")
		}

		if p.tok == COMMA {
			assignmentNode.AddChild(p.newNode(NodeComma))
			err = p.Next()
			if err != nil {
				return errs.Wrap(err, "Failed to scan")
			}
		}
	}

	var msg string
	switch p.tok {
	case STRING:
		err = p.ParseString(assignmentNode)
		msg = "Failed to parse string"
	case IDENTIFIER:
		err = p.ParseIdentifier(assignmentNode)
		msg = "Failed to parse identifier"
	case IN:
		// If we encounter an IN token then we've reached the end of the binding
		return nil
	default:
		err = p.ParseList(assignmentNode)
		msg = "Failed to parse list"
	}
	if err != nil {
		return errs.Wrap(err, msg)
	}

	return p.expectToken(R_CURL, assignmentNode, NodeRightCurlyBracket)
}

func (p *Parser) ParseIdentifier(node *Node) error {
	if !p.IsIdentifier() {
		return errs.New("Expected identifier, got: %s", p.tok.String())
	}

	identifierNode := p.newNode(NodeIdentifier)
	node.AddChild(identifierNode)

	return p.Next()
}

func (p *Parser) ParseApplication(node *Node) error {
	if !p.IsFunctionIdentifier() {
		return errs.New("Expected function identifier, got: %s, lit: %s", p.tok.String(), p.lit)
	}

	applicationNode := p.newNode(NodeApplication)
	node.AddChild(applicationNode)

	err := p.ParseFunctionIdentifier(applicationNode)
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

	err = p.ParseArguments(applicationNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse arguments")
	}

	return p.expectToken(R_CURL, applicationNode, NodeRightCurlyBracket)
}

func (p *Parser) ParseFunctionIdentifier(node *Node) error {
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

	return p.Next()
}

func (p *Parser) ParseArguments(node *Node) error {
	for {
		if !p.IsIdentifier() {
			break
		}

		err := p.ParseIdentifier(node)
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
			err = p.ParseString(node)
			msg = "Failed to parse string"
		case IDENTIFIER:
			err = p.ParseIdentifier(node)
			msg = "Failed to parse identifier"
		default:
			err = p.ParseList(node)
			msg = "Failed to parse list from arguments"
		}
		if err != nil {
			return errs.Wrap(err, msg)
		}

		if p.tok == COMMA {
			node.AddChild(p.newNode(NodeComma))
			err := p.Next()
			if err != nil {
				return errs.Wrap(err, "Failed to scan")
			}
			continue
		}
	}

	return nil
}

func (p *Parser) ParseList(node *Node) error {
	if p.tok != L_BRACKET {
		return errs.New("Expected left bracket, got: %s@pos:%d:%d", p.lit, p.pos.Column, p.pos.Line)
	}

	listNode := &Node{
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

	return p.expectToken(R_BRACKET, listNode, NodeRightBracket)
}

func (p *Parser) ParseListElements(node *Node) error {
	for {
		if p.tok != L_CURL && !p.IsString() && p.tok != COMMA {
			break
		}

		if p.tok == COMMA {
			node.AddChild(p.newNode(NodeComma))

			err := p.Next()
			if err != nil {
				return errs.Wrap(err, "Failed to scan")
			}
			continue
		}

		var err error
		var msg string
		switch p.tok {
		case L_CURL:
			err = p.ParseListObject(node)
			msg = "Failed to parse list object"
		case STRING:
			err = p.ParseListString(node)
			msg = "Failed to parse list string"
		}
		if err != nil {
			return errs.Wrap(err, msg)
		}

		if p.tok == R_BRACKET {
			node.AddChild(p.newNode(NodeRightBracket))
			break
		}
	}

	return nil
}

func (p *Parser) ParseListString(node *Node) error {
	elementNode := p.newNode(NodeListElement)
	node.AddChild(elementNode)

	return p.ParseString(elementNode)
}

func (p *Parser) ParseListObject(node *Node) error {
	elementNode := p.newNode(NodeListElement)
	node.AddChild(elementNode)

	err := p.expectToken(L_CURL, elementNode, NodeLeftCurlyBracket)
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	for {
		if p.tok != IDENTIFIER {
			break
		}

		err := p.ParseObjectAttribute(elementNode)
		if err != nil {
			return errs.Wrap(err, "Failed to parse object attribute")
		}

		if p.tok == COMMA {
			elementNode.AddChild(p.newNode(NodeComma))
			err := p.Next()
			if err != nil {
				return errs.Wrap(err, "Failed to scan")
			}
		}
	}

	return p.expectToken(R_CURL, elementNode, NodeRightCurlyBracket)
}

func (p *Parser) ParseObjectAttribute(node *Node) error {
	if p.tok != IDENTIFIER {
		return errs.New("Expected identifier")
	}

	err := p.ParseIdentifier(node)
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
		err = p.ParseString(node)
		msg = "Failed to parse string"
	case L_CURL:
		err = p.ParseListObject(node)
		msg = "Failed to parse list object"
	default:
		err = p.ParseList(node)
		msg = "Failed to parse list"
	}
	if err != nil {
		return errs.Wrap(err, msg)
	}

	return nil
}

func (p *Parser) ParseListElement(node *Node) error {
	elementNode := p.newNode(NodeListElement)
	node.AddChild(elementNode)

	switch p.tok {
	case STRING:
		return p.ParseString(elementNode)
	default:
		return errs.New("Expected string or identifier, got: %s", p.lit)
	}
}

func (p *Parser) ParseString(node *Node) error {
	if p.tok != STRING {
		return errs.New("Expected string")
	}
	node.AddChild(p.newNode(NodeString))

	return p.Next()
}

func (p *Parser) ParseIn(node *Node) error {
	if p.tok != IN {
		return errs.New("Expected in")
	}
	node.AddChild(p.newNode(NodeIn))

	err := p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	err = p.expectToken(COLON, node, NodeColon)
	if err != nil {
		return errs.Wrap(err, "Expect failed")
	}

	var msg string
	switch p.tok {
	case STRING:
		err = p.ParseString(node)
		msg = "Failed to parse string"
	case IDENTIFIER:
		err = p.ParseApplication(node)
		msg = "Failed to parse application"
	default:
		return errs.New("Expected string or identifier, got: %s", p.lit)
	}
	if err != nil {
		return errs.Wrap(err, msg)
	}

	return p.Next()
}

func (p *Parser) IsExpression() bool {
	// Currently only supporting Let, can be expanded to
	// support application and identifier later
	return p.tok == LET
}

func (p *Parser) IsIdentifier() bool {
	return p.tok == IDENTIFIER
}

func (p *Parser) IsFunctionIdentifier() bool {
	// TODO: This should be generalized to support all functions rather than just the ones we support
	// Can likely be done by using peek() to check for the next token
	return p.tok == F_SOLVE || p.tok == F_SOLVELEGACY || p.tok == F_MERGE
}

func (p *Parser) IsString() bool {
	return p.tok == STRING
}
