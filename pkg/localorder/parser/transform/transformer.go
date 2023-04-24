package transform

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/localorder/parser"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
)

type BuildScriptTransformer struct {
	root    *parser.NodeElement
	result  *BuildScript
	visited map[*parser.NodeElement]bool
	// TODO: Both of these could be replaced with a call stack
	inLet             bool
	currentIdentifier string
}

func NewBuildScriptTransformer(ast *parser.Tree) *BuildScriptTransformer {
	return &BuildScriptTransformer{
		root: ast.Root,
		result: &BuildScript{
			Let: make(map[string]Binding),
		},
		visited: make(map[*parser.NodeElement]bool),
	}
}

// GetCurrentNode returns the current node of the resulting BuildScript.
func (t *BuildScriptTransformer) GetCurrentNode() interface{} {
	// Returns the current node set by AddNode.
	return nil
}

// AddNode adds a node to the BuildScript result.
func (t *BuildScriptTransformer) AddNode(node interface{}) error {
	// This funciton is called everytime we add to the resulting BuildScript.
	// As a side effect, this also updates the current node that we are working on.
	return nil
}

func (t *BuildScriptTransformer) Transform() (*BuildScript, error) {
	return t.transformFile(t.root)
}

// TODO: With the position information, we could walk the tree rather
// than these more specific functions.

func (t *BuildScriptTransformer) transformFile(node *parser.NodeElement) (*BuildScript, error) {
	if node.Type() != parser.NodeFile {
		return nil, errs.New("Unexpected node type in transformFile: %s", node.Type().String())
	}

	// Expression
	for _, c := range node.Children() {
		switch c.Type() {
		case parser.NodeExpression:
			err := t.TransformExpression(c)
			if err != nil {
				return nil, errs.Wrap(err, "Failed to transform expression")
			}
		default:
			return nil, errs.New("Unexpected node type in transformFile: %s", c.Type().String())
		}
	}

	return t.result, nil
}

func (t *BuildScriptTransformer) TransformExpression(node *parser.NodeElement) error {
	if node.Type() != parser.NodeExpression {
		return errs.New("Unexpected node type in transformFile: %s", node.Type().String())
	}

	var identifier string
	for i, c := range node.Children() {
		switch c.Type() {
		case parser.NodeLet:
			t.inLet = true
			continue
		case parser.NodeBinding:
			err := t.TransformBinding(c)
			if err != nil {
				return errs.Wrap(err, "Failed to transform binding")
			}
		case parser.NodeIdentifier:
			// This is only valid if we are in the in clause of a let statement.
			identifier = c.Literal()
		case parser.NodeIn:
			err := t.TransformIn(c, node.Children()[i:])
			if err != nil {
				return errs.Wrap(err, "Failed to transform in")
			}
			t.inLet = false
		case parser.NodeApplication:
			// This is only valid if we are not in the in clause of a let statement.
			err := t.TransformApplication(c, identifier)
			if err != nil {
				return errs.Wrap(err, "Failed to transform application")
			}
		case parser.NodeColon:
			continue
		default:
			return errs.New("Unexpected node type in transformExpression: %s", c.Type().String())
		}
	}

	return nil
}

func (t *BuildScriptTransformer) TransformIn(node *parser.NodeElement, siblings []*parser.NodeElement) error {
	// TODO: This is a bit messy, but it works for now however with the current
	// map structure it can only handle identifiers.
	// This could be a potential for generics...
	if len(siblings) < 3 {
		return errs.New("Expected at least 3 siblings")
	}

	switch siblings[2].Type() {
	case parser.NodeIdentifier:
		t.result.In = InIdentifier(siblings[2].Literal())
	case parser.NodeApplication:
		err := t.TransformApplication(siblings[2], "")
		if err != nil {
			return errs.Wrap(err, "Failed to transform application")
		}
	default:
		return errs.New("Unexpected node type in transformIn: %s", siblings[2].Type().String())
	}

	return nil
}

func (t *BuildScriptTransformer) TransformBinding(node *parser.NodeElement) error {
	for _, c := range node.Children() {
		if c.Type() != parser.NodeAssignment {
			return errs.New("Unexpected binding child type: %s", c.Type().String())
		}

		err := t.TransformAssignment(c)
		if err != nil {
			return errs.Wrap(err, "Failed to transform assignment")
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformAssignment(node *parser.NodeElement) error {
	var identifier string
	for _, c := range node.Children() {
		fmt.Println("identifier: ", identifier)
		fmt.Println("Current Identifier: ", t.currentIdentifier)
		fmt.Println("Child type: ", c.Type().String())
		switch c.Type() {
		case parser.NodeIdentifier:
			identifier = c.Literal()
		case parser.NodeApplication:
			err := t.TransformApplication(c, identifier)
			if err != nil {
				return errs.Wrap(err, "Failed to transform application")
			}
		case parser.NodeList:
			err := t.TransformList(c, identifier)
			if err != nil {
				return errs.Wrap(err, "Failed to transform list")
			}
		case parser.NodeString:
			err := t.TransformString(c, identifier)
			if err != nil {
				return errs.Wrap(err, "Failed to transform string")
			}
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformApplication(applicationNode *parser.NodeElement, identifier string) error {
	for _, c := range applicationNode.Children() {
		switch c.Type() {
		case parser.NodeFunction:
			err := t.TransformFunction(c, identifier)
			if err != nil {
				return errs.Wrap(err, "Failed to transform function")
			}
		case parser.NodeBinding:
			err := t.TransformBinding(c)
			if err != nil {
				return errs.Wrap(err, "Failed to transform argument")
			}
		case parser.NodeLeftParen, parser.NodeRightParen:
			continue
		}
		t.currentIdentifier = identifier
		t.visited[c] = true
	}

	t.visited[applicationNode] = true
	t.currentIdentifier = ""
	return nil
}

func (t *BuildScriptTransformer) TransformFunction(node *parser.NodeElement, identifier string) error {
	for _, c := range node.Children() {
		switch c.Type() {
		case parser.NodeSolveFn, parser.NodeSolveLegacyFn:
			if t.inLet {
				t.result.Let[identifier] = SolveBinding{
					Platforms:    make([]string, 0),
					Requirements: make([]Requirement, 0),
				}
			} else {
				return errs.New("Cannot set SolveBinding on non-let statement")
			}
		case parser.NodeMergeFn:
			if t.inLet {
				return errs.New("Cannot set MergeApplication on let statement")
			}
			t.result.In = make(MergeApplication)
		case parser.NodeAppendFn:
			solveBinding, ok := t.result.Let[t.currentIdentifier].(SolveBinding)
			if !ok {
				return errs.New("Cannot append to non-SolveBinding")
			}
			solveBinding.Requirements = append(solveBinding.Requirements)
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformArgument(result *model.BuildScript, node *parser.NodeElement) error {
	if result.Let == nil {
		return errs.New("Cannot set argument on nil LetStatement")
	}
	if result.Let.Runtime == nil {
		return errs.New("Cannot set argument on nil Runtime")
	}
	if result.Let.Runtime.SolveLegacy == nil {
		return errs.New("Cannot set argument on nil SolveLegacy")
	}

	for _, c := range node.Children() {
		switch c.Type() {
		// TODO: Move the binding case to its own function
		case parser.NodeBinding:
			switch node.Literal() {
			case "platforms":
				result.Let.Runtime.SolveLegacy.Platforms = make([]string, 0)
			case "requirements":
				result.Let.Runtime.SolveLegacy.Requirements = make([]*model.Requirement, 0)
			default:
				return errs.New("Unexpected argument: %s", node.Literal())
			}
		default:
			return errs.New("Unexpected node in type in TransforArgument: %s, lit: %s", c.Type().String(), c.Literal())
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformList(node *parser.NodeElement, identifier string) error {
	for _, c := range node.Children() {
		switch c.Type() {
		case parser.NodeListElement:
			err := t.TransformListElement(c, identifier)
			if err != nil {
				return errs.Wrap(err, "Failed to transform string")
			}
		case parser.NodeRightBracket, parser.NodeLeftBracket, parser.NodeComma:
			continue
		default:
			return errs.New("Unexpected node type in TransformList: %s, lit: %s", c.Type().String(), c.Literal())
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformListElement(node *parser.NodeElement, identifier string) error {
	// This function will likely also need a position slice
	for _, c := range node.Children() {
		switch c.Type() {
		case parser.NodeString:
			switch identifier {
			case "platforms":
				if t.currentIdentifier == "" {
					return errs.New("Cannot set platform outside of a solve function")
				}
				solveBinding, ok := t.result.Let[t.currentIdentifier].(SolveBinding)
				if !ok {
					return errs.New("Cannot set platform on non-solve function")
				}
				solveBinding.Platforms = append(solveBinding.Platforms, c.Literal())
				t.result.Let[t.currentIdentifier] = solveBinding
			case "requirements":
				if t.currentIdentifier == "" {
					return errs.New("Cannot set requirement outside of a solve function")
				}
				solveBinding, ok := t.result.Let[t.currentIdentifier].(SolveBinding)
				if !ok {
					return errs.New("Cannot set requirement on non-solve function")
				}
				solveBinding.Requirements = append(solveBinding.Requirements, Requirement{
					Name: c.Literal(),
				})
				t.result.Let[t.currentIdentifier] = solveBinding
			default:
				return errs.New("TransformListElement(NodeString): Unexpected identifier: %s", identifier)
			}
		case parser.NodeIdentifier:
			switch t.result.Let[t.currentIdentifier].(type) {
			case SolveBinding:
				err := t.TransformReqFunc(c, identifier, t.result.Let[t.currentIdentifier].(SolveBinding))
				if err != nil {
					return errs.Wrap(err, "Failed to transform req function")
				}
			default:
				// TODO: Inspect current result here to debug
				return errs.New("TransformListElement(NodeIdentifier): Unexpected identifier: %s", identifier)
			}
		default:
			return errs.New("Unexpected node type in TransformListElement: %s, lit: %s", c.Type().String(), c.Literal())
		}
		t.visited[c] = true
	}

	return nil
}

func (t BuildScriptTransformer) TransformReqFunc(node *parser.NodeElement, identifier string, binding SolveBinding) error {
	for _, c := range node.Children() {
		if c.Type() != parser.NodeBinding {
			continue
		}
		for _, cc := range c.Children() {
			if cc.Type() != parser.NodeAssignment {
				continue
			}

			req := Requirement{}
			var identifier string
			for _, ccc := range cc.Children() {
				switch ccc.Type() {
				case parser.NodeIdentifier:
					switch ccc.Literal() {
					case "name":
						identifier = ccc.Literal()
					case "version":
						identifier = ccc.Literal()
					default:
						return errs.New("Unexpected identifier: %s", ccc.Literal())
					}
				case parser.NodeString:
					switch identifier {
					case "name":
						req.Name = ccc.Literal()
					case "version":
						versionReq := make(map[string]string)
						versionReq["eq"] = ccc.Literal()
						req.Version = versionReq
					default:
						return errs.New("Unexpected identifier: %s", ccc.Literal())
					}
					binding.Requirements = append(binding.Requirements, req)
					t.result.Let[t.currentIdentifier] = binding
				}
			}
		}
	}

	return nil
}

func (t *BuildScriptTransformer) TransformString(node *parser.NodeElement, identifier string) error {
	// TODO: Hanlde ONLY strings
	return nil
}
