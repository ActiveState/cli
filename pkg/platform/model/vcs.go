package model

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	vcsClient "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
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
	// FailNoCommitID indicates that no commit id is provided and not
	// obtainable from the current project.
	FailNoCommitID = failures.Type("languages.fail.nocommitid", failures.FailNonFatal)
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

	if branch.CommitID == nil {
		return nil, failures.FailUserInput.New(locale.Tl(
			"err_project_no_commit",
			"Your project does not have any commits yet, head over to https://{{.V0}}/{{.V1}}/{{.V2}} to set up your project.", constants.PlatformURL, ownerName, projectName))
	}

	return branch.CommitID, nil
}

// CommitHistory will return the commit history for the given owner / project
func CommitHistory(ownerName, projectName string) ([]*mono_models.Commit, *failures.Failure) {
	offset := int64(0)
	limit := int64(100)
	var commits []*mono_models.Commit

	cont := true
	for cont {
		payload, fail := CommitHistoryPaged(ownerName, projectName, offset, limit)
		if fail != nil {
			return commits, fail
		}
		commits = append(commits, payload.Commits...)
		cont = payload.TotalCommits > (offset + limit)
	}

	return commits, nil
}

// CommitHistory will return the commit history for the given owner / project
func CommitHistoryPaged(ownerName, projectName string, offset, limit int64) (*mono_models.CommitHistoryInfo, *failures.Failure) {
	latestCID, fail := LatestCommitID(ownerName, projectName)
	if fail != nil {
		return nil, fail
	}

	params := vcsClient.NewGetCommitHistoryParams()
	params.SetCommitID(*latestCID)
	params.Limit = &limit
	params.Offset = &offset
	res, err := authentication.Client().VersionControl.GetCommitHistory(params, authentication.ClientAuth())
	if err != nil {
		return nil, FailGetCommitHistory.New(locale.Tr("err_get_commit_history", api.ErrorMessageFromPayload(err)))
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

	indexed := makeIndexedCommits(res.Payload.Commits)
	return indexed.countBetween(commitID, latestCID.String())
}

// Changeset aliases for eased usage and to act as a disconnect from the underlying dep.
type Changeset = []*mono_models.CommitChangeEditable

// AddChangeset creates a new commit with multiple changes as provided. This is lower level than CommitChangeset.
func AddChangeset(parentCommitID strfmt.UUID, commitMessage string, changeset Changeset) (*mono_models.Commit, *failures.Failure) {
	params := vcsClient.NewAddCommitParams()
	params.SetCommit(&mono_models.CommitEditable{
		Changeset:      changeset,
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

// AddCommit creates a new commit with a single change. This is lower level than Commit{X} functions.
func AddCommit(parentCommitID strfmt.UUID, commitMessage string, operation Operation, namespace Namespace, requirement string, version string) (*mono_models.Commit, *failures.Failure) {
	changeset := []*mono_models.CommitChangeEditable{
		{
			Operation:         string(operation),
			Namespace:         string(namespace),
			Requirement:       requirement,
			VersionConstraint: version,
		},
	}

	return AddChangeset(parentCommitID, commitMessage, changeset)
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
		operation, NamespacePackage(languages[0].Name),
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

// CommitChangeset commits multiple changes in one commit
func CommitChangeset(projOwner, projName, commitMsg string, changeset Changeset) *failures.Failure {
	branch, fail := DefaultBranchForProjectName(projOwner, projName)
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

	commit, fail := AddChangeset(*branch.CommitID, commitMsg, changeset)
	if fail != nil {
		return fail
	}

	return UpdateBranchCommit(branch.BranchID, commit.CommitID)
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

// CommitPlatform commits a single platform commit
func CommitPlatform(owner, prjName string, op Operation, name, version string, word int) *failures.Failure {
	platform, fail := FetchPlatformByDetails(name, version, word)
	if fail != nil {
		return fail
	}

	proj, fail := FetchProjectByName(owner, prjName)
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

	var msgL10nKey string
	switch op {
	case OperationAdded:
		msgL10nKey = "commit_message_add_platform"
	case OperationUpdated:
		return failures.FailDeveloper.New("this is not supported yet")
	case OperationRemoved:
		msgL10nKey = "commit_message_removed_platform"
	}

	bCommitID := *branch.CommitID
	msg := locale.Tr(msgL10nKey, name, strconv.Itoa(word), version)
	platformID := platform.PlatformID.String()

	// version is not the value that AddCommit needs - platforms do not post a version
	commit, fail := AddCommit(bCommitID, msg, op, NamespacePlatform(), platformID, "")
	if fail != nil {
		return fail
	}

	return UpdateBranchCommit(branch.BranchID, commit.CommitID)
}

// CommitLanguage commits a single language to the platform
func CommitLanguage(owner, project string, op Operation, name, version string) *failures.Failure {
	lang, fail := FetchLanguageByDetails(name, version)
	if fail != nil {
		return fail
	}

	proj, fail := FetchProjectByName(owner, project)
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

	var msgL10nKey string
	switch op {
	case OperationAdded:
		msgL10nKey = "commit_message_add_language"
	case OperationUpdated:
		return failures.FailDeveloper.New("this is not supported yet")
	case OperationRemoved:
		msgL10nKey = "commit_message_removed_language"
	}

	branchCommitID := *branch.CommitID
	msg := locale.Tr(msgL10nKey, name, version)

	commit, fail := AddCommit(branchCommitID, msg, op, NamespaceLanguage(), lang.Name, lang.Version)
	if fail != nil {
		return fail
	}

	return UpdateBranchCommit(branch.BranchID, commit.CommitID)
}

func ChangesetFromRequirements(op Operation, reqs Checkpoint) Changeset {
	var changeset Changeset

	for _, req := range reqs {
		change := &mono_models.CommitChangeEditable{
			Operation:         string(op),
			Namespace:         req.Namespace,
			Requirement:       req.Requirement,
			VersionConstraint: req.VersionConstraint,
		}

		changeset = append(changeset, change)
	}

	return changeset
}

// FetchOrderFromCommit retrieves an order from a given commit ID
func FetchOrderFromCommit(commitID strfmt.UUID) (*mono_models.Order, error) {
	retry := retryhttp.New(retryhttp.DefaultClient)
	defer retry.Close()

	params := vcsClient.NewGetOrderParamsWithContext(retry.Context)
	params.SetHTTPClient(retry.Client.StandardClient())
	params.CommitID = commitID

	var res *vcsClient.GetOrderOK
	var err error
	if auth.Get().Authenticated() {
		res, err = mono.New().VersionControl.GetOrder(params, authentication.ClientAuth())
	} else {
		// Allow activation of public projects if user is not authenticated
		res, err = mono.New().VersionControl.GetOrder(params, nil)
	}
	if err != nil {
		return nil, errors.New(api.ErrorMessageFromPayload(err))
	}

	return res.Payload, err
}

func TrackBranch(source, target *mono_models.Project) *failures.Failure {
	sourceBranch, fail := DefaultBranchForProject(source)
	if fail != nil {
		return fail
	}

	targetBranch, fail := DefaultBranchForProject(target)
	if fail != nil {
		return fail
	}

	trackingType := mono_models.BranchEditableTrackingTypeNotify

	updateParams := vcsClient.NewUpdateBranchParams()
	branch := &mono_models.BranchEditable{
		TrackingType: &trackingType,
		Tracks:       &sourceBranch.BranchID,
	}
	updateParams.SetBranch(branch)
	updateParams.SetBranchID(targetBranch.BranchID)

	_, err := authentication.Client().VersionControl.UpdateBranch(updateParams, authentication.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		return api.FailUnknown.Wrap(err, msg)
	}
	return nil
}
