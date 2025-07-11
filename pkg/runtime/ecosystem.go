package runtime

import "github.com/ActiveState/cli/pkg/buildplan"

// eg.
// availableEcosystems = []func() (ecosystem, error){
// 	func() (ecosystem, error) {
// 		return &python.Ecosystem{}, nil
// 	},
// }
var availableEcosystems []func() ecosystem

type ecosystem interface {
	Init(runtimePath string, buildplan *buildplan.BuildPlan) error
	Namespaces() []string
	Add(artifact *buildplan.Artifact, artifactSrcPath string) ([]string, error)
	Remove(artifact *buildplan.Artifact) error
	Apply() error
}

func artifactMatchesEcosystem(a *buildplan.Artifact, e ecosystem) bool {
	for _, namespace := range e.Namespaces() {
		for _, i := range a.Ingredients {
			if i.Namespace == namespace {
				return true
			}
		}
	}
	return false
}

func namespacesMatchesEcosystem(namespaces []string, e ecosystem) bool {
	for _, namespace := range e.Namespaces() {
		for _, n := range namespaces {
			if n == namespace {
				return true
			}
		}
	}
	return false
}

func filterEcosystemMatchingArtifact(artifact *buildplan.Artifact, ecosystems []ecosystem) ecosystem {
	for _, ecosystem := range ecosystems {
		if artifactMatchesEcosystem(artifact, ecosystem) {
			return ecosystem
		}
	}
	return nil
}

func filterEcosystemsMatchingNamespaces(ecosystems []ecosystem, namespaces []string) []ecosystem {
	result := []ecosystem{}
	for _, ecosystem := range ecosystems {
		if namespacesMatchesEcosystem(namespaces, ecosystem) {
			result = append(result, ecosystem)
		}
	}
	return result
}
