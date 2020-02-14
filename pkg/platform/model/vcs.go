package model

import (
	"fmt"
	"regexp"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	vcsClient "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	// FailGetCommitHistory is a failure in the call to api.GetCommitHistory
	FailGetCommitHistory = failures.Type("model.fail.getcommithistory", failures.FailNonFatal)
	// FailCommitCountImpossible is a failure counting between commits
	FailCommitCountImpossible = failures.Type("model.fail.commitcountimpossible", failures.FailNonFatal)
	// FailCommitCountUnknowable is a failure counting between commits
	FailCommitCountUnknowable = failures.Type("model.fail.commitcountunknowable", failures.FailNonFatal)
	// FailAddCommit is a failure in adding a new commit
	FailAddCommit = failures.Type("model.fail.addcommit")
	// FailUpdateBranch is a failure in updating a branch
	FailUpdateBranch = failures.Type("model.fail.updatebranch")
	// FailNoCommit is a failure due to a non-existent commit
	FailNoCommit = failures.Type("model.fail.nocommit")
	// FailNoLanguages is a failure due to the checkpoint not having any languages
	FailNoLanguages = failures.Type("model.fail.nolanguages")
)

// Operation is the action to be taken in a commit
type Operation string

const (
	// OperationAdded is for adding a new requirement
	OperationAdded = Operation(mono_models.CommitChangeEditableOperationAdded)
	// OperationUpdated is for updating an existing requirement
	OperationUpdated = Operation(mono_models.CommitChangeEditableOperationUpdated)
	// OperationRemoved is for removing an existing requirement
	OperationRemoved = Operation(mono_models.CommitChangeEditableOperationRemoved)
)

// NamespaceMatchable represents regular expression strings used for defining matchable
// requirements.
type NamespaceMatchable string

const (
	// NamespacePlatformMatch is the namespace used for platform requirements
	NamespacePlatformMatch NamespaceMatchable = `^platform$`

	// NamespaceLanguageMatch is the namespace used for language requirements
	NamespaceLanguageMatch = `^language$`

	// NamespacePackageMatch is the namespace used for package requirements
	NamespacePackageMatch = `^language\/\w+$`

	// NamespacePrePlatformMatch is the namespace used for pre-platform bits
	NamespacePrePlatformMatch = `^pre-platform-installer$`

	// NamespaceCamelFlagsMatch is the namespace used for passing camel flags
	NamespaceCamelFlagsMatch = `^camel-flags$`
)

// NamespaceMatch Checks if the given namespace query matches the given namespace
func NamespaceMatch(query string, namespace NamespaceMatchable) bool {
	match, err := regexp.Match(string(namespace), []byte(query))
	if err != nil {
		logging.Error("Could not match regex for %v, query: %s, error: %v", namespace, query, err)
	}
	return match
}

// Namespace is the type used for communicating namespaces, mainly just allows for self documenting code
type Namespace string

// NamespacePackage creates a new package namespace
func NamespacePackage(language string) Namespace {
	return Namespace(fmt.Sprintf("language/%s", language))
}

// NamespaceLanguage provides the base language namespace.
func NamespaceLanguage() Namespace {
	return Namespace("language")
}

// NamespacePlatform provides the base platform namespace.
func NamespacePlatform() Namespace {
	return Namespace("platform")
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

// CommitHistory will return the commit history for the given owner / project
func CommitHistory(ownerName, projectName string) ([]*mono_models.Commit, *failures.Failure) {
	latestCID, fail := LatestCommitID(ownerName, projectName)
	if fail != nil {
		return nil, fail
	}

	params := vcsClient.NewGetCommitHistoryParams()
	params.SetCommitID(*latestCID)
	res, err := authentication.Client().VersionControl.GetCommitHistory(params, authentication.ClientAuth())
	if err != nil {
		return nil, FailGetCommitHistory.New(locale.Tr("err_get_commit_history", err.Error()))
	}

	return res.Payload, nil
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
		return 0, FailCommitCountImpossible.New(locale.T("err_commit_count_no_latest_with_commit"))
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

// AddCommit creates a new commit with a single change
func AddCommit(parentCommitID strfmt.UUID, commitMessage string, operation Operation, namespace Namespace, requirement string, version string) (*mono_models.Commit, *failures.Failure) {
	params := vcsClient.NewAddCommitParams()
	params.SetCommit(&mono_models.CommitEditable{
		Changeset: []*mono_models.CommitChangeEditable{&mono_models.CommitChangeEditable{
			Operation:         string(operation),
			Namespace:         string(namespace),
			Requirement:       requirement,
			VersionConstraint: version,
		}},
		Message:        commitMessage,
		ParentCommitID: parentCommitID,
	})

	res, err := authentication.Client().VersionControl.AddCommit(params, authentication.ClientAuth())
	if err != nil {
		logging.Error("AddCommit Error: %s", err.Error())
		return nil, FailAddCommit.New(locale.Tr("err_add_commit", api.ErrorMessageFromPayload(err)))
	}
	return res.Payload, nil
}

// UpdateBranchCommit updates the commit that a branch is pointed at
func UpdateBranchCommit(branchID strfmt.UUID, commitID strfmt.UUID) *failures.Failure {
	params := vcsClient.NewUpdateBranchParams()
	params.SetBranchID(branchID)
	params.SetBranch(&mono_models.BranchEditable{
		CommitID: &commitID,
	})

	_, err := authentication.Client().VersionControl.UpdateBranch(params, authentication.ClientAuth())
	if err != nil {
		return FailUpdateBranch.New(locale.Tr("err_update_branch", err.Error()))
	}
	return nil
}

// CommitPackage commits a single package commit
func CommitPackage(projectOwner, projectName string, operation Operation, packageName, packageVersion string) *failures.Failure {
	proj, fail := FetchProjectByName(projectOwner, projectName)
	if fail != nil {
		return fail
	}

	branch, fail := DefaultBranchForProject(proj)
	if fail != nil {
		return fail
	}

	if branch.CommitID == nil {
		return FailNoCommit.New(locale.T("err_project_no_languages"))
	}

	languages, fail := FetchLanguagesForCommit(*branch.CommitID)
	if fail != nil {
		return fail
	}

	if len(languages) == 0 {
		return FailNoLanguages.New(locale.T("err_project_no_languages"))
	}

	var message string
	switch operation {
	case OperationAdded:
		message = "commit_message_add_package"
	case OperationUpdated:
		message = "commit_message_updated_package"
	case OperationRemoved:
		message = "commit_message_removed_package"
	}

	commit, fail := AddCommit(*branch.CommitID, locale.Tr(message, packageName, packageVersion),
		operation, NamespacePackage(languages[0]),
		packageName, packageVersion)
	if fail != nil {
		return fail
	}

	fail = UpdateBranchCommit(branch.BranchID, commit.CommitID)
	if fail != nil {
		return fail
	}

	return nil
}

// CommitInitial ...
func CommitInitial(projectOwner, projectName, hostPlatform, language, langVersion string) (*mono_models.Project, strfmt.UUID, *failures.Failure) {
	platformID, fail := hostPlatformToPlatformID(hostPlatform)
	if fail != nil {
		return nil, "", fail
	}

	proj, fail := FetchProjectByName(projectOwner, projectName)
	if fail != nil {
		return nil, "", fail
	}

	branch, fail := DefaultBranchForProject(proj)
	if fail != nil {
		return nil, "", fail
	}

	if branch.CommitID != nil {
		return nil, "", FailUpdateBranch.New(locale.T("err_branch_not_bare"))
	}

	var changes []*mono_models.CommitChangeEditable

	if language != "" {
		c := &mono_models.CommitChangeEditable{
			Operation:         string(OperationAdded),
			Namespace:         string(NamespaceLanguage()),
			Requirement:       language,
			VersionConstraint: langVersion,
		}
		changes = append(changes, c)
	}

	c := &mono_models.CommitChangeEditable{
		Operation:         string(OperationAdded),
		Namespace:         string(NamespacePlatform()),
		Requirement:       platformID,
		VersionConstraint: "",
	}
	changes = append(changes, c)

	commit := &mono_models.CommitEditable{
		Changeset: changes,
		Message:   locale.T("commit_message_add_initial"),
	}
	params := vcsClient.NewAddCommitParams()
	params.SetCommit(commit)

	res, err := authentication.Client().VersionControl.AddCommit(params, authentication.ClientAuth())
	if err != nil {
		logging.Error("AddCommit Error: %s", err.Error())
		return nil, "", FailAddCommit.New(locale.Tr("err_add_commit", api.ErrorMessageFromPayload(err)))
	}

	fail = UpdateBranchCommit(branch.BranchID, res.Payload.CommitID)
	if fail != nil {
		return nil, "", fail
	}

	return proj, res.Payload.CommitID, nil
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
		return 0, FailCommitCountImpossible.New(locale.T("err_commit_count_missing_last"))
	}

	if first != "" {
		if _, ok := cs[first]; !ok {
			return 0, FailCommitCountUnknowable.New(locale.Tr("err_commit_count_cannot_find_first", first))
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
			return 0, FailCommitCountUnknowable.New(locale.Tr("err_commit_count_cannot_find", next))
		}
	}

	return ct, nil
}
