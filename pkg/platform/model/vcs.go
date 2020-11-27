package model

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
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

	// NamespacePackageMatch is the namespace used for language package requirements
	NamespacePackageMatch = `^language\/\w+$`

	// NamespaceBundlesMatch is the namespace used for bundle package requirements
	NamespaceBundlesMatch = `^bundles\/\w+$`

	// NamespacePrePlatformMatch is the namespace used for pre-platform bits
	NamespacePrePlatformMatch = `^pre-platform-installer$`

	// NamespaceCamelFlagsMatch is the namespace used for passing camel flags
	NamespaceCamelFlagsMatch = `^camel-flags$`
)

// NamespacePrefix is set to a prefix for ingredient namespaces in the inventory
type NamespacePrefix string

const (
	// PackageNamespacePrefix is the namespace prefix for packages
	PackageNamespacePrefix NamespacePrefix = "language"

	// BundlesNamespacePrefix is the namespace prefix for bundles
	BundlesNamespacePrefix = "bundles"
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

// NamespaceBundles creates a new bundles namespace
func NamespaceBundles(language string) Namespace {
	return Namespace(fmt.Sprintf("bundles/%s", language))
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
func CommitHistory(ownerName, projectName string) ([]*mono_models.Commit, error) {
	latestCID, fail := LatestCommitID(ownerName, projectName)
	if fail != nil {
		return nil, fail.ToError()
	}
	return commitHistory(*latestCID)
}

// CommitHistoryFromID will return the commit history from the given commitID
func CommitHistoryFromID(commitID strfmt.UUID) ([]*mono_models.Commit, error) {
	return commitHistory(commitID)
}

func commitHistory(commitID strfmt.UUID) ([]*mono_models.Commit, error) {
	offset := int64(0)
	limit := int64(100)
	var commits []*mono_models.Commit

	cont := true
	for cont {
		payload, fail := CommitHistoryPaged(commitID, offset, limit)
		if fail != nil {
			return commits, fail.ToError()
		}
		commits = append(commits, payload.Commits...)
		cont = payload.TotalCommits > (offset + limit)
	}

	return commits, nil
}

// CommitHistoryPaged will return the commit history for the given owner / project
func CommitHistoryPaged(commitID strfmt.UUID, offset, limit int64) (*mono_models.CommitHistoryInfo, *failures.Failure) {
	params := vcsClient.NewGetCommitHistoryParams()
	params.SetCommitID(commitID)
	params.Limit = &limit
	params.Offset = &offset

	var res *vcsClient.GetCommitHistoryOK
	var err error
	if authentication.Get().Authenticated() {
		res, err = authentication.Client().VersionControl.GetCommitHistory(params, authentication.ClientAuth())
	} else {
		res, err = mono.New().VersionControl.GetCommitHistory(params, nil)
	}
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
func AddChangeset(parentCommitID strfmt.UUID, commitMessage string, isHeadless bool, changeset Changeset) (*mono_models.Commit, *failures.Failure) {
	params := vcsClient.NewAddCommitParams()

	commit := &mono_models.CommitEditable{
		Changeset:      changeset,
		Message:        commitMessage,
		ParentCommitID: parentCommitID,
	}

	if isHeadless {
		id := logging.UniqID()
		commit.AnonID = &id
	}

	params.SetCommit(commit)

	res, err := mono.New().VersionControl.AddCommit(params, authentication.ClientAuth())
	if err != nil {
		logging.Error("AddCommit Error: %s", err.Error())
		return nil, FailAddCommit.New(locale.Tr("err_add_commit", api.ErrorMessageFromPayload(err)))
	}
	return res.Payload, nil
}

// AddCommit creates a new commit with a single change. This is lower level than Commit{X} functions.
func AddCommit(parentCommitID strfmt.UUID, commitMessage string, operation Operation, namespace Namespace, requirement string, version string, isHeadless bool) (*mono_models.Commit, *failures.Failure) {
	changeset := []*mono_models.CommitChangeEditable{
		{
			Operation:         string(operation),
			Namespace:         string(namespace),
			Requirement:       requirement,
			VersionConstraint: version,
		},
	}

	return AddChangeset(parentCommitID, commitMessage, isHeadless, changeset)
}

// UpdateBranchCommit updates the commit that a branch is pointed at
func UpdateBranchCommit(branchID strfmt.UUID, commitID strfmt.UUID) error {
	params := vcsClient.NewUpdateBranchParams()
	params.SetBranchID(branchID)
	params.SetBranch(&mono_models.BranchEditable{
		CommitID: &commitID,
	})

	_, err := authentication.Client().VersionControl.UpdateBranch(params, authentication.ClientAuth())
	if err != nil {
		if _, ok := err.(*version_control.UpdateBranchForbidden); ok {
			err = locale.WrapError(
				err,
				"err_update_branch_permissions",
				"You do not have permission to modify the requirements for this project. You will either need to be invited to the project or you can fork it by running [ACTIONABLE]state fork <project namespace>[/RESET].",
			)
			return errs.AddTips(err, "Run [ACTIONABLE]state fork <project namespace>[/RESET] to make changes to this project")
		}
		return locale.NewError("err_update_branch", api.ErrorMessageFromPayload(err))
	}
	return nil
}

// CommitPackage commits a package to an existing parent commit
func CommitPackage(parentCommitID strfmt.UUID, operation Operation, packageName, packageNamespace, packageVersion string, isHeadless bool) (strfmt.UUID, *failures.Failure) {
	var commitID strfmt.UUID
	languages, fail := FetchLanguagesForCommit(parentCommitID)
	if fail != nil {
		return commitID, fail
	}

	if len(languages) == 0 {
		return commitID, FailNoLanguages.New(locale.T("err_project_no_languages"))
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

	namespace := NamespacePackage(languages[0].Name)
	if strings.HasPrefix(packageNamespace, string(BundlesNamespacePrefix)) {
		namespace = NamespaceBundles(languages[0].Name)
	}

	commit, fail := AddCommit(
		parentCommitID, locale.Tr(message, packageName, packageVersion),
		operation, namespace,
		packageName, packageVersion, isHeadless,
	)
	if fail != nil {
		return commitID, fail
	}
	return commit.CommitID, nil
}

// UpdateProjectBranchCommit updates the vcs brach for a given project with a new commitID
func UpdateProjectBranchCommit(proj *mono_models.Project, commitID strfmt.UUID) error {
	branch, fail := DefaultBranchForProject(proj)
	if fail != nil {
		return errs.Wrap(fail.ToError(), "Failed to get default branch for project %s.", proj.Name)
	}

	return UpdateBranchCommit(branch.BranchID, commitID)
}

// UpdateProjectBranchCommitByName updates the vcs branch for a project given by its namespace with a new commitID
func UpdateProjectBranchCommitByName(projectOwner, projectName string, commitID strfmt.UUID) error {
	proj, fail := FetchProjectByName(projectOwner, projectName)
	if fail != nil {
		return errs.Wrap(fail.ToError(), "Failed to fetch project.")
	}

	return UpdateProjectBranchCommit(proj, commitID)
}

// CommitChangeset commits multiple changes in one commit
func CommitChangeset(parentCommitID strfmt.UUID, commitMsg string, isHeadless bool, changeset Changeset) (strfmt.UUID, error) {
	var commitID strfmt.UUID
	languages, fail := FetchLanguagesForCommit(parentCommitID)
	if fail != nil {
		return commitID, fail.ToError()
	}

	if len(languages) == 0 {
		return commitID, FailNoLanguages.New(locale.T("err_project_no_languages")).ToError()
	}

	commit, fail := AddChangeset(parentCommitID, commitMsg, isHeadless, changeset)
	if fail != nil {
		return commitID, fail.ToError()
	}
	return commit.CommitID, nil
}

// CommitInitial creates a root commit for a new branch
func CommitInitial(hostPlatform string, lang *language.Supported, langVersion string) (strfmt.UUID, *failures.Failure) {
	var language string
	if lang != nil {
		language = lang.Requirement()
		if langVersion == "" {
			langVersion = lang.RecommendedVersion()
		}
	}

	platformID, fail := hostPlatformToPlatformID(hostPlatform)
	if fail != nil {
		return "", fail
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
		return "", FailAddCommit.New(locale.Tr("err_add_commit", api.ErrorMessageFromPayload(err)))
	}

	return res.Payload.CommitID, nil
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
func CommitPlatform(owner, prjName string, op Operation, name, version string, word int) error {
	platform, fail := FetchPlatformByDetails(name, version, word)
	if fail != nil {
		return fail.ToError()
	}

	proj, fail := FetchProjectByName(owner, prjName)
	if fail != nil {
		return fail.ToError()
	}

	branch, fail := DefaultBranchForProject(proj)
	if fail != nil {
		return fail.ToError()
	}

	if branch.CommitID == nil {
		return FailNoCommit.New(locale.T("err_project_no_languages")).ToError()
	}

	var msgL10nKey string
	switch op {
	case OperationAdded:
		msgL10nKey = "commit_message_add_platform"
	case OperationUpdated:
		return failures.FailDeveloper.New("this is not supported yet").ToError()
	case OperationRemoved:
		msgL10nKey = "commit_message_removed_platform"
	}

	bCommitID := *branch.CommitID
	msg := locale.Tr(msgL10nKey, name, strconv.Itoa(word), version)
	platformID := platform.PlatformID.String()

	// version is not the value that AddCommit needs - platforms do not post a version
	// TODO: Headless check for caller of this func?
	commit, fail := AddCommit(bCommitID, msg, op, NamespacePlatform(), platformID, "", false)
	if fail != nil {
		return fail.ToError()
	}

	return UpdateBranchCommit(branch.BranchID, commit.CommitID)
}

// CommitLanguage commits a single language to the platform
func CommitLanguage(owner, project string, op Operation, name, version string) error {
	lang, fail := FetchLanguageByDetails(name, version)
	if fail != nil {
		return fail.ToError()
	}

	proj, fail := FetchProjectByName(owner, project)
	if fail != nil {
		return fail.ToError()
	}

	branch, fail := DefaultBranchForProject(proj)
	if fail != nil {
		return fail.ToError()
	}

	if branch.CommitID == nil {
		return FailNoCommit.New(locale.T("err_project_no_languages")).ToError()
	}

	var msgL10nKey string
	switch op {
	case OperationAdded:
		msgL10nKey = "commit_message_add_language"
	case OperationUpdated:
		return failures.FailDeveloper.New("this is not supported yet").ToError()
	case OperationRemoved:
		msgL10nKey = "commit_message_removed_language"
	}

	branchCommitID := *branch.CommitID
	msg := locale.Tr(msgL10nKey, name, version)

	// TODO: Headless check for caller of this func?
	commit, fail := AddCommit(branchCommitID, msg, op, NamespaceLanguage(), lang.Name, lang.Version, false)
	if fail != nil {
		return fail.ToError()
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
	params := vcsClient.NewGetOrderParams()
	params.CommitID = commitID
	params.SetHTTPClient(retryhttp.DefaultClient.StandardClient())

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

func GetRevertCommit(from, to strfmt.UUID) (*mono_models.Commit, error) {
	params := vcsClient.NewGetRevertCommitParams()
	params.SetCommitFromID(from)
	params.SetCommitToID(to)

	client := mono.New()
	if authentication.Get().Authenticated() {
		client = authentication.Client()
	}
	res, err := client.VersionControl.GetRevertCommit(params, authentication.ClientAuth())
	if err != nil {
		return nil, locale.WrapError(err, "err_get_revert_commit", "Could not revert from commit ID {{.V0}} to {{.V1}}", from.String(), to.String())
	}

	return res.Payload, nil
}

func RevertCommit(owner, project string, from, to strfmt.UUID) error {
	revertCommit, err := GetRevertCommit(from, to)
	if err != nil {
		return err
	}

	addCommit, err := AddRevertCommit(revertCommit)
	if err != nil {
		return err
	}

	proj, fail := FetchProjectByName(owner, project)
	if fail != nil {
		return err
	}

	branch, fail := DefaultBranchForProject(proj)
	if fail != nil {
		return err
	}

	err = UpdateBranchCommit(branch.BranchID, addCommit.CommitID)
	if err != nil {
		return err
	}

	return nil
}

func GetCommit(commitID strfmt.UUID) (*mono_models.Commit, error) {
	params := vcsClient.NewGetCommitParams()
	params.SetCommitID(commitID)
	params.SetHTTPClient(retryhttp.DefaultClient.StandardClient())

	res, err := authentication.Client().VersionControl.GetCommit(params, authentication.ClientAuth())
	if err != nil {
		return nil, locale.WrapError(err, "err_get_commit", "Could not get commit from ID: {{.V0}}", commitID.String())
	}
	return res.Payload, nil
}

func AddRevertCommit(commit *mono_models.Commit) (*mono_models.Commit, error) {
	params := vcsClient.NewAddCommitParams()

	editableCommit, err := commitToCommitEditable(commit)
	if err != nil {
		return nil, locale.WrapError(err, "err_convert_commit", "Could not convert commit data")
	}
	params.SetCommit(editableCommit)

	res, err := authentication.Client().VersionControl.AddCommit(params, authentication.ClientAuth())
	if err != nil {
		return nil, locale.WrapError(err, "err_add_revert_commit", "Could not add revert commit")
	}
	return res.Payload, nil
}

func commitToCommitEditable(from *mono_models.Commit) (*mono_models.CommitEditable, error) {
	editableData, err := from.MarshalBinary()
	if err != nil {
		return nil, locale.WrapError(err, "err_commit_marshal", "Could not marshall commit data")
	}

	commit := &mono_models.CommitEditable{}
	err = commit.UnmarshalBinary(editableData)
	if err != nil {
		return nil, locale.WrapError(err, "err_commit_unmarshal", "Could not unmarshal commit data")
	}
	return commit, nil
}
