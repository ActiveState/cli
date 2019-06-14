package model

import (
	"fmt"
	"regexp"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	vcsClient "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// Namespace represents regular expression strings used for defining matchable
// requirements.
type Namespace string

const (
	// NamespacePlatform is the namespace used for platform requirements
	NamespacePlatform Namespace = `^platform$`

	// NamespaceLanguage is the namespace used for language requirements
	NamespaceLanguage = `^language$`

	// NamespacePackage is the namespace used for package requirements
	NamespacePackage = `/package$`
)

// NamespaceMatch Checks if the given namespace query matches the given namespace
func NamespaceMatch(query string, namespace Namespace) bool {
	match, err := regexp.Match(string(namespace), []byte(query))
	if err != nil {
		logging.Error("Could not match regex for %v, query: %s, error: %v", namespace, query, err)
	}
	return match
}

// LatestCommitID returns the latest commit id by owner and project names. It
// possible for a nil commit id to be returned without failure.
func LatestCommitID(ownerName, projectName string) (*strfmt.UUID, *failures.Failure) {
	proj, fail := FetchProjectByName(ownerName, projectName)
	if fail != nil {
		return nil, fail
	}

	branch, fail := DefaultBranchForProject(proj)
	if fail != nil {
		return nil, fail
	}

	return branch.CommitID, nil
}

// CommitsBehindLatest compares the provided commit id with the latest commit
// id and returns the count of commits it is behind.
func CommitsBehindLatest(ownerName, projectName, commitID string) (int, *failures.Failure) {
	latestCID, fail := LatestCommitID(ownerName, projectName)
	if fail != nil {
		return 0, fail
	}

	if latestCID == nil {
		if commitID == "" {
			return 0, nil // ok, nothing to do
		}
		return 0, nil // special fail (commitID with no latest)
	}

	params := vcsClient.NewGetCommitHistoryParams()
	params.SetCommitID(*latestCID)
	res, err := authentication.Client().VersionControl.GetCommitHistory(params, authentication.ClientAuth())
	if err != nil {
		return 0, nil // wrap error with failure
	}

	ordered := makeOrderedCommits(res.Payload)
	ct, err := ordered.countBetween(commitID, latestCID.String())
	if err != nil {
		return ct, nil // wrap error with failure
	}

	return ct, nil
}

type orderedCommits map[string]string // key == commit id / val == parent id

func makeOrderedCommits(cs []*mono_models.Commit) orderedCommits {
	m := make(orderedCommits)

	for _, c := range cs {
		m[string(c.CommitID)] = string(c.ParentCommitID)
	}

	return m
}

func (cs orderedCommits) countBetween(first, last string) (int, error) {
	next := last
	var ok bool
	var ct int

	for next != "" {
		next, ok = cs[next]
		if !ok {
			efmt := "cannot find commit %q in history"
			return ct, fmt.Errorf(efmt, next)
		}
		ct++
	}

	return ct, nil
}
