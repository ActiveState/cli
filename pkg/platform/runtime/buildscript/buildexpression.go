package buildscript

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/p"
	buildexpression "github.com/ActiveState/cli/pkg/buildexpression/parser"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
)

func FromBuildExpression(tree *buildexpression.Tree) *Script {
	script := &Script{&Let{}, &In{}}
	// tree.Root.Children() contains the following 7 nodes: { let : <binding> , in <in expr>

	letBinding := tree.Root.Children()[1].Children()[3]
	for _, node := range letBinding.Children() {
		script.Let.Assignments = append(script.Let.Assignments, fromAssignment(node))
	}

	switch inExpr := tree.Root.Children()[1].Children()[6]; inExpr.Type() {
	case buildexpression.NodeApplication:
		script.In.FuncCall = fromApplication(inExpr)
	case buildexpression.NodeString:
		name := strings.TrimLeft(inExpr.Literal(), "$")
		script.In.Name = &name
	default:
		logging.Error("Unknown inExpr type: %v", inExpr.Type())
	}

	return script
}

func fromAssignment(node *buildexpression.Node) *Assignment {
	logging.Debug("Evaluating NodeAssignment")
	children := node.Children()
	assignment := &Assignment{Key: children[0].Literal()}
	switch children[2].Type() {
	case buildexpression.NodeLeftCurlyBracket:
		assignment.Value = &Value{FuncCall: fromApplication(children[3])}
	default:
		logging.Error("Unhandled assignment node: %v", children[2])
	}
	return assignment
}

func fromApplication(node *buildexpression.Node) *FuncCall {
	logging.Debug("Evaluating NodeApplication")
	children := node.Children()
	funcCall := &FuncCall{Name: children[0].Children()[0].Literal()}
	// NodeApplication's tree is weird. Its "function parameters" are inlined and consist of a
	// variable number of nodes. Lookahead is needed to produce a correct Value{}.
	i := 3 // skip the following nodes: <func> : {
	for i < len(children) {
		child := children[i]
		i++
		if child.Type() == buildexpression.NodeComma {
			continue // skip to next node
		} else if child.Type() == buildexpression.NodeRightCurlyBracket {
			break // done
		}
		value := &Value{}
		switch children[i].Type() { // i is always a valid index
		case buildexpression.NodeColon:
			i++
			assignment := &Assignment{Key: child.Literal()}
			switch valueNode := children[i]; valueNode.Type() {
			case buildexpression.NodeList:
				assignment.Value = &Value{List: fromList(valueNode)}
			case buildexpression.NodeString:
				assignment.Value = &Value{String: fromString(valueNode)}
			case buildexpression.NodeIdentifier:
				assignment.Value = &Value{Ident: p.StrP(valueNode.Literal())}
			default:
				logging.Error("Unhandled assignment node: %v", valueNode)
			}
			i++
			value.Assignment = assignment
		default:
			value.Ident = p.StrP(child.Literal())
		}
		funcCall.Arguments = append(funcCall.Arguments, value)
	}
	return funcCall
}

func fromList(node *buildexpression.Node) *[]*Value {
	logging.Debug("Evaluating NodeList")
	values := []*Value{}
	// ListNode contains not only NodeListElement node, but also syntax nodes like [ and ,
	// Fortunately, we can ignore everything but NodeListElement.
	for _, child := range node.Children() {
		if child.Type() != buildexpression.NodeListElement {
			continue
		}
		children := child.Children()
		value := &Value{}
		switch children[0].Type() {
		case buildexpression.NodeLeftCurlyBracket:
			assignments := []*Assignment{}
			// Each assignment consists of 4 inlined nodes: <key> : <value> ,
			for i := 1; i < len(children); i += 4 {
				assignment := &Assignment{Key: children[i].Literal()}
				switch valueNode := children[i+2]; valueNode.Type() {
				case buildexpression.NodeString:
					assignment.Value = &Value{String: fromString(valueNode)}
				case buildexpression.NodeList:
					assignment.Value = &Value{List: fromList(valueNode)}
				default:
					assignment.Value = &Value{Ident: p.StrP(valueNode.Literal())}
				}
				assignments = append(assignments, assignment)
			}
			value.Object = &assignments
		case buildexpression.NodeString:
			value.String = fromString(children[0])
		default:
			value.Ident = p.StrP(children[0].Literal())
		}
		values = append(values, value)
	}
	return &values
}

func fromString(node *buildexpression.Node) *string {
	// This node does not retain the quotes, so add them back.
	s := fmt.Sprintf(`"%s"`, node.Literal())
	return &s
}

func (s *Script) ToBuildExpression() (*buildexpression.Tree, error) {
	parser, err := buildexpression.New(s.ToJson())
	if err != nil {
		return nil, errs.Wrap(err, "Unable to create build expression parser")
	}
	tree, err := parser.Parse()
	if err != nil {
		return nil, errs.Wrap(err, "Unable to parse JSON")
	}
	return tree, nil
}

func (s *Script) EqualsBuildExpression(other *buildexpression.Tree) bool {
	myJson := string(s.ToJson())
	otherJson := string(FromBuildExpression(other).ToJson())
	return myJson == otherJson
}

func (s *Script) Equals(other *model.BuildScript) bool { return false } // TODO
