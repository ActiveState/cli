package buildplan

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/go-openapi/strfmt"
)

type walkFunc func(walkNodeContext) error

func (b *RawBuild) walkNodes(nodeIDs []strfmt.UUID, walk walkFunc) error {
	lookup := make(map[strfmt.UUID]interface{})

	for _, artifact := range b.Artifacts {
		lookup[artifact.NodeID] = artifact
	}
	for _, step := range b.Steps {
		lookup[step.StepID] = step
	}
	for _, source := range b.Sources {
		lookup[source.NodeID] = source
	}

	for _, nodeID := range nodeIDs {
		node, ok := lookup[nodeID]
		if !ok {
			return errs.New("node ID '%s' does not exist in lookup table", nodeID)
		}
		if err := walkNode(walkNodeContext{
			node:           node,
			parentArtifact: nil,
			lookup:         lookup,
		}, walk); err != nil {
			return errs.Wrap(err, "could not recurse over node IDs")
		}
	}

	return nil
}

type walkNodeContext struct {
	node              interface{}
	parentArtifact    *Artifact
	lookup            map[strfmt.UUID]interface{}
	isBuildDependency bool // Whether we are in a dependency step; ie. a build time dependency
}

func walkNode(w walkNodeContext, walk walkFunc) error {
	if err := walk(w); err != nil {
		return errs.Wrap(err, "error walking over node")
	}

	source, ok := w.node.(*Source)
	if ok {
		return nil // Sources are at the end of the recursion.
	}

	ar, ok := w.node.(*Artifact)
	if !ok {
		return errs.New("node ID '%v' is not an artifact", w.node)
	}

	generatedByNode, ok := w.lookup[ar.GeneratedBy]
	if !ok {
		return nil
	}

	// Sources can also be referenced by the generatedBy property
	source, ok = generatedByNode.(*Source)
	if ok {
		if err := walk(walkNodeContext{
			node:              source,
			parentArtifact:    ar,
			isBuildDependency: w.isBuildDependency,
			lookup:            w.lookup,
		}); err != nil {
			return errs.Wrap(err, "error walking over source")
		}
		return nil // Sources are at the end of the recursion.
	}

	step, ok := generatedByNode.(*Step)
	if !ok {
		return errs.New("node ID '%s' has unexpected type '%T'", ar.GeneratedBy, generatedByNode)
	}

	for _, input := range step.Inputs {
		if input.Tag != TagSource && input.Tag != TagDependency {
			continue
		}
		for _, id := range input.NodeIDs {
			subNode, ok := w.lookup[id]
			if !ok {
				return errs.New("node ID '%s' does not exist in lookup table", id)
			}
			if err := walkNode(walkNodeContext{
				node:              subNode,
				parentArtifact:    ar,
				isBuildDependency: w.isBuildDependency || input.Tag == TagDependency,
				lookup:            w.lookup,
			}, walk); err != nil {
				return errs.Wrap(err, "error iterating over %s", id)
			}
		}
	}

	return nil
}
