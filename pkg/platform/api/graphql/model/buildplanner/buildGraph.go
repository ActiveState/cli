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
func NewBuildScript() *BuildScript {
	return &BuildScript{
		Let: LetStatement{
			Runtime: Runtime{
				SolveLegacy: SolveLegacy{
					Requirements: []Requirement{},
				},
			},
		},
	}
}

type BuildScript struct {
	Let LetStatement `json:"let"`
}

type LetStatement struct {
	Runtime Runtime `json:"runtime"`
}

type Runtime struct {
	// TODO: Will this also need a solve field?
	SolveLegacy SolveLegacy `json:"solve_legacy"`
}

type SolveLegacy struct {
	Requirements []Requirement `json:"requirements"`
}

type Requirement struct {
	Name               string               `json:"name"`
	Namespace          string               `json:"namespace"`
	VersionRequirement []VersionRequirement `json:"version_requirements,omitempty"`
}

type VersionRequirement map[Comparator]string

func (bs *BuildScript) Update(operation Operation, requirements []Requirement) (*BuildScript, error) {
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

func (bg *BuildScript) add(requirements []Requirement) *BuildScript {
	bg.Let.Runtime.SolveLegacy.Requirements = append(bg.Let.Runtime.SolveLegacy.Requirements, requirements...)
	return bg
}

func (bg *BuildScript) remove(requirements []Requirement) *BuildScript {
	for i, req := range bg.Let.Runtime.SolveLegacy.Requirements {
		for _, removeReq := range requirements {
			if req.Name == removeReq.Name && req.Namespace == removeReq.Namespace {
				bg.Let.Runtime.SolveLegacy.Requirements = append(bg.Let.Runtime.SolveLegacy.Requirements[:i], bg.Let.Runtime.SolveLegacy.Requirements[i+1:]...)
			}
		}
	}
	return bg
}

func (bg *BuildScript) update(requirements []Requirement) *BuildScript {
	for _, req := range bg.Let.Runtime.SolveLegacy.Requirements {
		for _, updateReq := range requirements {
			if req.Name == updateReq.Name && req.Namespace == updateReq.Namespace {
				req.VersionRequirement = updateReq.VersionRequirement
			}
		}
	}
	return bg
}
