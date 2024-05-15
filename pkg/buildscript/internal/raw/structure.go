package raw

import (
	"time"
)

// Raw 's tagged fields will be initially filled in by Participle.
// expr will be constructed later and is this script's buildexpression. We keep a copy of the build
// expression here with any changes that have been applied before either writing it to disk or
// submitting it to the build planner. It's easier to operate on build expressions directly than to
// modify or manually populate the Participle-produced fields and re-generate a build expression.
type Raw struct {
	Assignments []*Assignment `parser:"@@+"`
	AtTime      *time.Time
}

type Assignment struct {
	Key   string `parser:"@Ident '='"`
	Value *Value `parser:"@@"`
}

type Value struct {
	FuncCall *FuncCall `parser:"@@"`
	List     *[]*Value `parser:"| '[' (@@ (',' @@)* ','?)? ']'"`
	Str      *string   `parser:"| @String"`
	Number   *float64  `parser:"| (@Float | @Int)"`
	Null     *Null     `parser:"| @@"`

	Assignment *Assignment    `parser:"| @@"`                        // only in FuncCall
	Object     *[]*Assignment `parser:"| '{' @@ (',' @@)* ','? '}'"` // only in List
	Ident      *string        `parser:"| @Ident"`                    // only in FuncCall or Assignment
}

type Null struct {
	Null string `parser:"'null'"`
}

type FuncCall struct {
	Name      string   `parser:"@Ident"`
	Arguments []*Value `parser:"'(' @@ (',' @@)* ','? ')'"`
}

type In struct {
	FuncCall *FuncCall `parser:"@@"`
	Name     *string   `parser:"| @Ident"`
}

var (
	reqFuncName = "Req"
	eqFuncName  = "Eq"
	neFuncName  = "Ne"
	gtFuncName  = "Gt"
	gteFuncName = "Gte"
	ltFuncName  = "Lt"
	lteFuncName = "Lte"
	andFuncName = "And"
)
