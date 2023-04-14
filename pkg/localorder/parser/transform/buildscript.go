package transform

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/localorder/parser"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
)

type BuildScriptTransformer struct {
	ast     *parser.Tree
	visited map[*parser.NodeElement]bool
}

type transformFunc func(*model.BuildScript, *parser.NodeElement) error

func NewBuildScriptTransformer(ast *parser.Tree) *BuildScriptTransformer {
	return &BuildScriptTransformer{
		ast:     ast,
		visited: make(map[*parser.NodeElement]bool),
	}
}

func (t *BuildScriptTransformer) Transform2() (*model.BuildScript, error) {
	result := model.NewBuildScript()

	err := t.walkTree(t.ast, result, t.ast.Root)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to walk tree")
	}

	return result, nil
}

func (t *BuildScriptTransformer) walkTree(tree *parser.Tree, result *model.BuildScript, node *parser.NodeElement) error {
	for _, child := range node.Children() {
		switch nodeType := child.Type(); {
		case nodeType == parser.NodeLet:
			err := t.TransformExpression(result, child)
			if err != nil {
				return errs.Wrap(err, "Failed to transform expression")
			}
		case nodeType == parser.NodeIdentifier:
			err := t.TransformIdentifier(result, child)
			if err != nil {
				return errs.Wrap(err, "Failed to transform identifier")
			}
		case nodeType == parser.NodeApplication:
			err := t.TransformApplication(result, child)
			if err != nil {
				return errs.Wrap(err, "Failed to transform application")
			}
		case nodeType == parser.NodeBinding:
			err := t.TransformBinding(result, child)
			if err != nil {
				return errs.Wrap(err, "Failed to transform binding")
			}
		case nodeType == parser.NodeIn:
			err := t.TransformIn(result, child)
			if err != nil {
				return errs.Wrap(err, "Failed to transform in")
			}
		case nodeType == parser.NodeArgument:
			err := t.TransformArgument(result, child)
			if err != nil {
				return errs.Wrap(err, "Failed to transform argument")
			}
		case t.skippableNode(child):
			continue
		default:
			return errs.New("Unexpected node type: %s", nodeType.String())
		}
		t.visited[child] = true

		err := t.walkTree(tree, result, child)
		if err != nil {
			return errs.Wrap(err, "Failed to walk tree")
		}
	}

	return nil
}

func (t *BuildScriptTransformer) TransformExpression(result *model.BuildScript, node *parser.NodeElement) error {
	if result.Let == nil {
		return errs.New("Cannot set runtime on nil LetStatement")
	}
	return nil
}

func (t *BuildScriptTransformer) TransformIdentifier(result *model.BuildScript, identifierNode *parser.NodeElement) error {
	if result.Let == nil {
		return errs.New("Cannot set runtime on nil LetStatement")
	}

	// TODO: Do we need to do anything here?

	return nil
}

func (t *BuildScriptTransformer) TransformApplication(result *model.BuildScript, applicationNode *parser.NodeElement) error {
	if result.Let == nil {
		return errs.New("Cannot set application on nil LetStatement")
	}
	if result.Let.Runtime == nil {
		return errs.New("Cannot set application on nil Runtime")
	}

	for _, c := range applicationNode.Children() {
		switch c.Type() {
		case parser.NodeFunction:
			err := t.TransformFunction(result, c)
			if err != nil {
				return errs.Wrap(err, "Failed to transform function")
			}
		case parser.NodeBinding:
			err := t.TransformBinding(result, c)
			if err != nil {
				return errs.Wrap(err, "Failed to transform argument")
			}
		case parser.NodeLeftParen, parser.NodeRightParen:
			continue
		}
		t.visited[c] = true
	}

	t.visited[applicationNode] = true
	return nil
}

func (t *BuildScriptTransformer) TransformFunction(result *model.BuildScript, node *parser.NodeElement) error {
	if result.Let == nil {
		return errs.New("Cannot set application on nil LetStatement")
	}
	if result.Let.Runtime == nil {
		return errs.New("Cannot set application on nil Runtime")
	}

	for _, c := range node.Children() {
		if c.Type() == parser.NodeSolveFn || c.Type() == parser.NodeSolveLegacyFn {
			// Only supporting solve_legacy for now
			result.Let.Runtime.SolveLegacy = &model.SolveLegacy{}
		} else {
			return errs.New("Unexpected function child type: %s", c.Type().String())
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformIn(result *model.BuildScript, node *parser.NodeElement) error {
	if result.Let == nil {
		return errs.New("Cannot set in on nil LetStatement")
	}
	result.Let.In = "in"
	return nil
}

func (t *BuildScriptTransformer) TransformBinding(result *model.BuildScript, node *parser.NodeElement) error {
	if result.Let == nil {
		return errs.New("Cannot set binding on nil LetStatement")
	}

	for _, c := range node.Children() {
		switch c.Type() {
		case parser.NodeAssignment:
			err := t.TransformAssignment(result, c)
			if err != nil {
				return errs.Wrap(err, "Failed to transform assignment")
			}
		case parser.NodeList:
			// err := t.TransformList(result, c)
			// if err != nil {
			// 	return errs.Wrap(err, "Failed to transform list")
			// }
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformAssignment(result *model.BuildScript, node *parser.NodeElement) error {
	if result.Let == nil {
		return errs.New("Cannot set argument on nil LetStatement")
	}

	var identifier string
	for _, c := range node.Children() {
		switch c.Type() {
		case parser.NodeIdentifier:
			err := t.TransformIdentifier(result, c)
			if err != nil {
				return errs.Wrap(err, "Failed to transform identifier")
			}
			identifier = c.Literal()
		case parser.NodeApplication:
			err := t.TransformApplication(result, c)
			if err != nil {
				return errs.Wrap(err, "Failed to transform application")
			}
		case parser.NodeList:
			err := t.TransformList(result, c, identifier)
			if err != nil {
				return errs.Wrap(err, "Failed to transform list")
			}
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
			return errs.New("Unexpected node type: %s", c.Type().String())
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformList(result *model.BuildScript, node *parser.NodeElement, identifier string) error {
	if result.Let == nil {
		return errs.New("Cannot set list on nil LetStatement")
	}
	if result.Let.Runtime == nil {
		return errs.New("Cannot set list on nil Runtime")
	}
	if result.Let.Runtime.SolveLegacy == nil {
		return errs.New("Cannot set list on nil SolveLegacy")
	}

	for _, c := range node.Children() {
		switch c.Type() {
		case parser.NodeListElement:
			err := t.TransformString(result, c, identifier)
			if err != nil {
				return errs.Wrap(err, "Failed to transform string")
			}
		case parser.NodeRightBracket, parser.NodeLeftBracket, parser.NodeComma:
			continue
		default:
			return errs.New("Unexpected node type: %s", c.Type().String())
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) TransformString(result *model.BuildScript, node *parser.NodeElement, identifier string) error {
	if result.Let == nil {
		return errs.New("Cannot set string on nil LetStatement")
	}
	if result.Let.Runtime == nil {
		return errs.New("Cannot set string on nil Runtime")
	}
	if result.Let.Runtime.SolveLegacy == nil {
		return errs.New("Cannot set string on nil SolveLegacy")
	}

	for _, c := range node.Children() {
		switch c.Type() {
		case parser.NodeString:
			switch identifier {
			case "platforms":
				result.Let.Runtime.SolveLegacy.Platforms = append(result.Let.Runtime.SolveLegacy.Platforms, c.Literal())
			case "requirements", "languages", "packages":
				result.Let.Runtime.SolveLegacy.Requirements = append(result.Let.Runtime.SolveLegacy.Requirements, &model.Requirement{
					Name: c.Literal(),
				})
			default:
				return errs.New("Unexpected list identifier: %s", identifier)
			}
		default:
			return errs.New("Unexpected node type: %s", c.Type().String())
		}
		t.visited[c] = true
	}

	return nil
}

func (t *BuildScriptTransformer) skippableNode(node *parser.NodeElement) bool {
	nt := node.Type()
	return t.visited[node] || nt == parser.NodeComment || nt == parser.NodeColon
}
