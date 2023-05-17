package parser

import (
	"fmt"

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

	// Should be the start of the JSON expression
	if p.tok != L_CURL {
		return nil, errs.New("Expected left curly bracket")
	}

	err = p.Next()
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
	expressionNode := p.newNode(NodeExpression)
	root.AddChild(expressionNode)

	expressionNode.AddChild(p.newNode(NodeLet))

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

	if p.tok != L_CURL {
		return errs.New("Expected left curly bracket")
	}

	err = p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.tok != IN {
		return errs.New("Expected in, got: %s, %s", p.lit, p.tok)
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

	if p.IsFunctionIdentifier() {
		return p.ParseApplication(expressionNode)
	}

	return nil
}

func (p *Parser) ParseBinding(node *NodeElement) error {
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
	return p.peek() == COLON
}

func (p *Parser) ParseAssignment(node *NodeElement) error {
	if !p.IsAssignment() {
		return errs.New("Expected assignment")
	}

	assignmentNode := p.newNode(NodeAssignment)
	node.AddChild(assignmentNode)

	if p.tok != COLON {
		// TODO: Need convenience error function
		return errs.New("Expected equals, got: type: %s, lit: %s", p.tok, p.lit)
	}
	assignmentNode.AddChild(p.newNode(NodeColon))

	err := p.Next()
	if err != nil {
		return errs.Wrap(err, "Failed to scan")
	}

	if p.IsFunctionIdentifier() {
		return p.ParseApplication(assignmentNode)
	}

	if p.tok == STRING {
		return p.ParseString(assignmentNode)
	}

	return p.ParseList(assignmentNode)
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

	err = p.ParseArguments(applicationNode)
	if err != nil {
		return errs.Wrap(err, "Failed to parse arguments")
	}

	return p.Next()
}

func (p *Parser) ParseFunctionIdentifier(node *NodeElement) error {
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

func (p *Parser) ParseArguments(node *NodeElement) error {
	for {
		if !p.IsFunctionIdentifier() {
			break
		}

		if p.IsFunctionIdentifier() {
			err := p.ParseApplication(node)
			if err != nil {
				return errs.Wrap(err, "Failed to parse application")
			}
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

func (p *Parser) ParseList(node *NodeElement) error {
	if p.tok != L_BRACKET {
		return errs.New("Expected left bracket, got: %s", p.lit)
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

	if p.tok != R_BRACKET {
		return errs.New("Expected right bracket, current lit: %s", p.lit)
	}
	listNode.AddChild(p.newNode(NodeRightBracket))

	return p.Next()
}

func (p *Parser) ParseListElements(node *NodeElement) error {
	for {
		if !p.IsFunctionIdentifier() && !p.IsString() && p.tok != COMMA {
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

		err := p.ParseListElement(node)
		if err != nil {
			return errs.Wrap(err, "Failed to parse list element")
		}

		if p.tok == R_BRACKET {
			node.AddChild(p.newNode(NodeRightBracket))
			break
		}
	}

	return nil
}

func (p *Parser) ParseListElement(node *NodeElement) error {
	elementNode := p.newNode(NodeListElement)
	node.AddChild(elementNode)

	switch p.tok {
	case STRING:
		return p.ParseString(elementNode)
	default:
		return errs.New("Expected string or identifier, got: %s", p.lit)
	}
}

func (p *Parser) ParseString(node *NodeElement) error {
	if p.tok != STRING {
		return errs.New("Expected string")
	}
	node.AddChild(p.newNode(NodeString))

	return p.Next()
}

func (p *Parser) IsExpression() bool {
	// Currently only supporting Let, can be expanded to
	// support application and identifier later
	fmt.Println("Token: ", p.tok)
	return p.tok == LET
}

func (p *Parser) IsFunctionIdentifier() bool {
	// TODO: This should be generalized to support all functions rather than just the ones we support
	// Can likely be done by using peek() to check for the next token
	return p.tok == F_SOLVE || p.tok == F_SOLVELEGACY || p.tok == F_MERGE
}

func (p *Parser) IsString() bool {
	return p.tok == STRING
}
