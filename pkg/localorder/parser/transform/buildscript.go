package transform

import (
	"github.com/ActiveState/cli/pkg/localorder/parser"
)

type BuildScriptTransformer struct {
	ast *parser.Tree
}

func NewBuildScriptTransformer(ast *parser.Tree) *BuildScriptTransformer {
	return &BuildScriptTransformer{
		ast: ast,
	}
}

func (t *BuildScriptTransformer) Transform() error {
	return nil
}
