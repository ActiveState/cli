package model

import "github.com/ActiveState/cli/internal/errs"

type Comparator string

type Operation int

const (
	ComparatorEQ  Comparator = "eq"
	ComparatorGT             = "gt"
	ComparatorGTE            = "gte"
	ComparatorLT             = "lt"
	ComparatorLTE            = "lte"
	ComparatorNE             = "ne"

	OperationAdd Operation = iota
	OperationRemove
	OperationUpdate
)

func (o Operation) String() string {
	switch o {
	case OperationAdd:
		return "add"
	case OperationRemove:
		return "remove"
	case OperationUpdate:
		return "update"
	default:
		return "unknown"
	}
}

// TODO: We will likely need some sort of parser or other solution for the build graph
func NewBuildExpression() *BuildExpression {
	return &BuildExpression{
		Let: LetStatement{
			Runtime: Runtime{
				SolveLegacy: SolveLegacy{
					Requirements: []Requirement{},
				},
			},
		},
	}
}

type BuildExpression struct {
	Let LetStatement `json:"let"`
}

type LetStatement struct {
	In      string  `json:"in"`
	Runtime Runtime `json:"runtime"`
}

type Runtime struct {
	Solve       Solve       `json:"solve,omitempty"`
	SolveLegacy SolveLegacy `json:"solve_legacy,omitempty"`
}

type Solve struct {
	BuildFlags    []string      `json:"build_flags,omitempty"`
	CamelFlags    []string      `json:"camel_flags,omitempty"`
	Platforms     []string      `json:"platforms"`
	SolverVersion string        `json:"solver_version,omitempty"`
	AtTime        string        `json:"at_time"`
	Requirements  []Requirement `json:"requirements"`
}

type SolveLegacy struct {
	BuildFlags    []string      `json:"build_flags,omitempty"`
	CamelFlags    []string      `json:"camel_flags,omitempty"`
	Platforms     []string      `json:"platforms"`
	SolverVersion string        `json:"solver_version,omitempty"`
	AtTime        string        `json:"at_time"`
	Requirements  []Requirement `json:"requirements"`
}

type Requirement struct {
	Name               string               `json:"name"`
	Namespace          string               `json:"namespace"`
	VersionRequirement []VersionRequirement `json:"version_requirements,omitempty"`
}

type VersionRequirement map[Comparator]string

func (bs *BuildExpression) Update(operation Operation, requirements []Requirement) (*BuildExpression, error) {
	switch operation {
	case OperationAdd:
		return bs.add(requirements), nil
	case OperationRemove:
		return bs.remove(requirements), nil
	case OperationUpdate:
		return bs.update(requirements), nil
	default:
		return nil, errs.New("Invalid operation")
	}
}

func (bs *BuildExpression) add(requirements []Requirement) *BuildExpression {
	bs.Let.Runtime.SolveLegacy.Requirements = append(bs.Let.Runtime.SolveLegacy.Requirements, requirements...)
	return bs
}

func (bs *BuildExpression) remove(requirements []Requirement) *BuildExpression {
	for i, req := range bs.Let.Runtime.SolveLegacy.Requirements {
		for _, removeReq := range requirements {
			if req.Name == removeReq.Name && req.Namespace == removeReq.Namespace {
				bs.Let.Runtime.SolveLegacy.Requirements = append(bs.Let.Runtime.SolveLegacy.Requirements[:i], bs.Let.Runtime.SolveLegacy.Requirements[i+1:]...)
			}
		}
	}
	return bs
}

func (bs *BuildExpression) update(requirements []Requirement) *BuildExpression {
	for _, req := range bs.Let.Runtime.SolveLegacy.Requirements {
		for _, updateReq := range requirements {
			if req.Name == updateReq.Name && req.Namespace == updateReq.Namespace {
				req.VersionRequirement = updateReq.VersionRequirement
			}
		}
	}
	return bs
}
