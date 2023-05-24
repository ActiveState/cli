package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/thoas/go-funk"
)

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
		Let: &LetStatement{
			Runtime: &Runtime{
				SolveLegacy: &SolveLegacy{
					Requirements: []*Requirement{},
				},
			},
		},
	}
}

// TODO: We may want to move this out of the model package
type BuildScript struct {
	Let *LetStatement `json:"let" yaml:"let"`
}

type LetStatement struct {
	In      string   `json:"in" yaml:"in"`
	Runtime *Runtime `json:"runtime" yaml:"runtime"`
}

type Runtime struct {
	// Solve       *Solve       `json:"solve,omitempty" yaml:"solve,omitempty"`
	SolveLegacy *SolveLegacy `json:"solve_legacy,omitempty" yaml:"solve_legacy,omitempty"`
}

type Solve struct {
	BuildFlags    []string       `json:"build_flags" yaml:"build_flags"`
	CamelFlags    []string       `json:"camel_flags" yaml:"camel_flags"`
	Platforms     []string       `json:"platforms" yaml:"platforms"`
	SolverVersion *string        `json:"solver_version" yaml:"solver_version"`
	AtTime        string         `json:"at_time" yaml:"at_time"`
	Requirements  []*Requirement `json:"requirements" yaml:"requirements"`
}

type SolveLegacy struct {
	BuildFlags    []string       `json:"build_flags" yaml:"build_flags"`
	CamelFlags    []string       `json:"camel_flags" yaml:"camel_flags"`
	Platforms     []string       `json:"platforms" yaml:"platforms"`
	SolverVersion *string        `json:"solver_version" yaml:"solver_version"`
	AtTime        string         `json:"at_time" yaml:"at_time"`
	Requirements  []*Requirement `json:"requirements" yaml:"requirements"`
}

type Requirement struct {
	Name               string                `json:"name" yaml:"name"`
	Namespace          string                `json:"namespace" yaml:"namespace"`
	VersionRequirement []*VersionRequirement `json:"version_requirements,omitempty" yaml:"version_requirements,omitempty"`
}

type VersionRequirement map[Comparator]string

func (bs *BuildScript) Update(operation Operation, requirements []*Requirement) (*BuildScript, error) {
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

func (bs *BuildScript) add(requirements []*Requirement) *BuildScript {
	bs.Let.Runtime.SolveLegacy.Requirements = append(bs.Let.Runtime.SolveLegacy.Requirements, requirements...)
	return bs
}

func (bs *BuildScript) remove(requirements []*Requirement) *BuildScript {
	for i, req := range bs.Let.Runtime.SolveLegacy.Requirements {
		for _, removeReq := range requirements {
			if req.Name == removeReq.Name && req.Namespace == removeReq.Namespace {
				bs.Let.Runtime.SolveLegacy.Requirements = append(bs.Let.Runtime.SolveLegacy.Requirements[:i], bs.Let.Runtime.SolveLegacy.Requirements[i+1:]...)
			}
		}
	}
	return bs
}

func (bs *BuildScript) update(requirements []*Requirement) *BuildScript {
	for _, req := range bs.Let.Runtime.SolveLegacy.Requirements {
		for _, updateReq := range requirements {
			if req.Name == updateReq.Name && req.Namespace == updateReq.Namespace {
				req.VersionRequirement = updateReq.VersionRequirement
			}
		}
	}
	return bs
}

// TODO: Verify this is the correct way to compare build scripts
func (bs *BuildScript) Equals(other *BuildScript) bool {
	if len(bs.Let.Runtime.SolveLegacy.Requirements) != len(other.Let.Runtime.SolveLegacy.Requirements) {
		return false
	}

	if !funk.Equal(bs.Let.Runtime.SolveLegacy.Platforms, other.Let.Runtime.SolveLegacy.Platforms) {
		return false
	}

	if !funk.Equal(bs.Let.Runtime.SolveLegacy.Requirements, other.Let.Runtime.SolveLegacy.Requirements) {
		return false
	}

	return true
}
