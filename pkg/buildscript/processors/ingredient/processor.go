package ingredient

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/buildscript"
)

type Processor struct {
	prime primeable
}

type primeable interface {
	primer.SvcModeler
	primer.Projecter
}

func NewProcessor(prime primeable) *Processor {
	return &Processor{prime}
}

func (p *Processor) FuncName() string {
	return "ingredient"
}

func (p *Processor) ToBuildExpression(script *buildscript.BuildScript, fn *buildscript.FuncCall) error {
	pj := p.prime.Project()
	if pj == nil {
		return errs.Wrap(rationalize.ErrNoProject, "Need project to hash globs (for cwd)")
	}

	var src *buildscript.Value
	for _, arg := range fn.Arguments {
		if arg.Assignment != nil && arg.Assignment.Key == "src" {
			src = arg.Assignment.Value
		}
	}
	if src == nil {
		return locale.NewInputError("err_marshalbuildexp_src_missing")
	}
	if src.List == nil {
		return locale.NewInputError("err_marshalbuildexp_src_invalid_type")
	}
	patterns := []string{}
	for _, value := range *src.List {
		if value.Str == nil {
			return locale.NewInputError("err_marshalbuildexp_src_item_invalid_type")
		}
		patterns = append(patterns, *value.Str)
	}

	hash, err := p.prime.SvcModel().HashGlobs(pj.Dir(), patterns)
	if err != nil {
		return errs.Wrap(err, "Could not hash globs")
	}

	fn.Arguments = append(fn.Arguments, &buildscript.Value{
		Assignment: &buildscript.Assignment{
			Key: "hash",
			Value: &buildscript.Value{
				Str: &hash,
			},
		},
	})

	return nil
}

func (p *Processor) FromBuildExpression(script *buildscript.BuildScript, call *buildscript.FuncCall) error {
	return nil
}
