package model

import (
	"fmt"
	"regexp"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	vcsClient "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	// FailGetCommitHistory is a failure in the call to api.GetCommitHistory
	FailGetCommitHistory = failures.Type("model.fail.getcommithistory")
	// FailCommitCountImpossible is a failure counting between commits
	FailCommitCountImpossible = failures.Type("model.fail.commitcountimpossible")
	// FailCommitCountUnknowable is a failure counting between commits
	FailCommitCountUnknowable = failures.Type("model.fail.commitcountunknowable")
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
// id and returns the count of commits it is behind. If an error is returned
// along with a value of -1, then the provided commit is more than likely
// behind, but it is not possible to clarify the count exactly.
func CommitsBehindLatest(ownerName, projectName, commitID string) (int, *failures.Failure) {
	latestCID, fail := LatestCommitID(ownerName, projectName)
	if fail != nil {
		return 0, fail
	}

	if latestCID == nil {
		if commitID == "" {
			return 0, nil // ok, nothing to do
		}
		return 0, FailCommitCountImpossible.New("latest commit id is not set while commit id is set")
	}

	if latestCID.String() == commitID {
		return 0, nil
	}

	params := vcsClient.NewGetCommitHistoryParams()
	params.SetCommitID(*latestCID)
	res, err := authentication.Client().VersionControl.GetCommitHistory(params, authentication.ClientAuth())
	if err != nil {
		return 0, FailGetCommitHistory.New(locale.Tr("err_get_commit_history", err.Error()))
	}

	indexed := makeIndexedCommits(res.Payload)
	return indexed.countBetween(commitID, latestCID.String())
}

type indexedCommits map[string]string // key == commit id / val == parent id

func makeIndexedCommits(cs []*mono_models.Commit) indexedCommits {
	m := make(indexedCommits)

	for _, c := range cs {
		m[string(c.CommitID)] = string(c.ParentCommitID)
	}

	return m
}

// countBetween returns 0 if same or if unable to determine the count. If the
// last commit is empty, -1 is returned. Caution: Currently, the logic does not
// verify that the first commit is "before" the last commit.
func (cs indexedCommits) countBetween(first, last string) (int, *failures.Failure) {
	if first == last {
		return 0, nil
	}

	if last == "" {
		return 0, FailCommitCountImpossible.New("missing last commit id")
	}

	if first != "" {
		if _, ok := cs[first]; !ok {
			return 0, FailCommitCountUnknowable.New("missing first commit id")
		}
	}

	next := last
	var ct int
	for ct <= len(cs) {
		if next == first {
			return ct, nil
		}

		ct++

		var ok bool
		next, ok = cs[next]
		if !ok {
			msg := fmt.Sprintf("cannot find commit (%s) in indexed", next)
			return 0, FailCommitCountUnknowable.New(msg)
		}
	}

	return ct, nil
}
