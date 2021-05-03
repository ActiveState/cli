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
	ErrCommitCountUnknowable = errs.New("Commit count is unknowable")
)

type ProjectInfo interface {
	Owner() string
	Name() string
	CommitUUID() strfmt.UUID
	BranchName() string
}

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

type TrackingType string

const (
	// TrackingNotify represents the notify tracking type for branches and will
	// notify the project owner of upstream changes
	TrackingNotify TrackingType = TrackingType(mono_models.BranchEditableTrackingTypeNotify)
	// TrackingIgnore represents the ignore tracking type for branches and will
	// ignore upstream changes
	TrackingIgnore = TrackingType(mono_models.BranchEditableTrackingTypeIgnore)
	// TrackingAutoUpdate represents the auto update tracking type for branches and will
	// auto update the branch with any upstream changes
	TrackingAutoUpdate = TrackingType(mono_models.BranchEditableTrackingTypeAutoUpdate)
)

func (t TrackingType) String() string {
	switch t {
	case TrackingNotify:
		return mono_models.BranchEditableTrackingTypeNotify
	case TrackingAutoUpdate:
		return mono_models.BranchEditableTrackingTypeAutoUpdate
	default:
		return mono_models.BranchEditableTrackingTypeIgnore
	}
}

// NamespaceMatch Checks if the given namespace query matches the given namespace
func NamespaceMatch(query string, namespace NamespaceMatchable) bool {
	match, err := regexp.Match(string(namespace), []byte(query))
	if err != nil {
		logging.Error("Could not match regex for %v, query: %s, error: %v", namespace, query, err)
	}
	return match
}

type NamespaceType struct {
	name   string
	prefix string
}

var (
	NamespacePackage  = NamespaceType{"package", "language"} // these values should match the namespace prefix
	NamespaceBundle   = NamespaceType{"bundle", "bundles"}
	NamespaceLanguage = NamespaceType{"language", ""}
	NamespacePlatform = NamespaceType{"platform", ""}
)

func (t NamespaceType) String() string {
	return t.name
}

func (t NamespaceType) Prefix() string {
	return t.prefix
}

// Namespace is the type used for communicating namespaces, mainly just allows for self documenting code
type Namespace struct {
	nsType NamespaceType
	value  string
}

func (n Namespace) Type() NamespaceType {
	return n.nsType
}

func (n Namespace) String() string {
	return n.value
}

func NewNamespacePkgOrBundle(language string, nstype NamespaceType) Namespace {
	if nstype == NamespaceBundle {
		return NewNamespaceBundle(language)
	}
	return NewNamespacePackage(language)
}

// NewNamespacePackage creates a new package namespace
func NewNamespacePackage(language string) Namespace {
	return Namespace{NamespacePackage, fmt.Sprintf("language/%s", language)}
}

// NewNamespaceBundle creates a new bundles namespace
func NewNamespaceBundle(language string) Namespace {
	return Namespace{NamespaceBundle, fmt.Sprintf("bundles/%s", language)}
}

// NewNamespaceLanguage provides the base language namespace.
func NewNamespaceLanguage() Namespace {
	return Namespace{NamespaceLanguage, "language"}
}

// NewNamespacePlatform provides the base platform namespace.
func NewNamespacePlatform() Namespace {
	return Namespace{NamespacePlatform, "platform"}
}

// BranchCommitID returns the latest commit id by owner and project names. It
// is possible for a nil commit id to be returned without failure.
func BranchCommitID(ownerName, projectName, branchName string) (*strfmt.UUID, error) {
	proj, err := FetchProjectByName(ownerName, projectName)
	if err != nil {
		return nil, err
	}

	branch, err := BranchForProjectByName(proj, branchName)
	if err != nil {
		return nil, err
	}

	if branch.CommitID == nil {
		return nil, locale.NewInputError(
			"err_project_no_commit",
			"Your project does not have any commits yet, head over to https://{{.V0}}/{{.V1}}/{{.V2}} to set up your project.", constants.PlatformURL, ownerName, projectName)
	}

	return branch.CommitID, nil
}

// CommitHistory will return the commit history for the given owner / project
func CommitHistory(ownerName, projectName, branchName string) ([]*mono_models.Commit, error) {
	latestCID, err := BranchCommitID(ownerName, projectName, branchName)
	if err != nil {
		return nil, err
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
		payload, err := CommitHistoryPaged(commitID, offset, limit)
		if err != nil {
			return commits, err
		}
		commits = append(commits, payload.Commits...)
		cont = payload.TotalCommits > (offset + limit)
	}

	return commits, nil
}

// CommitHistoryPaged will return the commit history for the given owner / project
func CommitHistoryPaged(commitID strfmt.UUID, offset, limit int64) (*mono_models.CommitHistoryInfo, error) {
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
		return nil, locale.WrapError(err, "err_get_commit_history", "", api.ErrorMessageFromPayload(err))
	}

	return res.Payload, nil
}

// CommitsBehind compares the provided commit id with the latest commit
// id and returns the count of commits it is behind. If an error is returned
// along with a value of -1, then the provided commit is more than likely
// behind, but it is not possible to clarify the count exactly.
func CommitsBehind(latestCID, currentCommitID strfmt.UUID) (int, error) {
	if latestCID == "" {
		if currentCommitID == "" {
			return 0, nil // ok, nothing to do
		}
		return 0, locale.NewError("err_commit_count_no_latest_with_commit")
	}

	if latestCID.String() == currentCommitID.String() {
		return 0, nil
	}

	params := vcsClient.NewGetCommitHistoryParams()
	params.SetCommitID(latestCID)
	res, err := authentication.Client().VersionControl.GetCommitHistory(params, authentication.ClientAuth())
	if err != nil {
		return 0, locale.WrapError(err, "err_get_commit_history", "", err.Error())
	}

	indexed := makeIndexedCommits(res.Payload.Commits)
	return indexed.countBetween(currentCommitID.String(), latestCID.String())
}

// Changeset aliases for eased usage and to act as a disconnect from the underlying dep.
type Changeset = []*mono_models.CommitChangeEditable

// AddChangeset creates a new commit with multiple changes as provided. This is lower level than CommitChangeset.
func AddChangeset(parentCommitID strfmt.UUID, commitMessage string, anonymousID string, changeset Changeset) (*mono_models.Commit, error) {
	params := vcsClient.NewAddCommitParams()

	commit := &mono_models.CommitEditable{
		Changeset:      changeset,
		Message:        commitMessage,
		ParentCommitID: parentCommitID,
		AnonID:         anonymousID,
	}

	params.SetCommit(commit)

	res, err := mono.New().VersionControl.AddCommit(params, authentication.ClientAuth())
	if err != nil {
		logging.Error("AddCommit Error: %s", err.Error())
		return nil, locale.WrapError(err, "err_add_commit", "", api.ErrorMessageFromPayload(err))
	}
	return res.Payload, nil
}

// AddCommit creates a new commit with a single change. This is lower level than Commit{X} functions.
func AddCommit(parentCommitID strfmt.UUID, commitMessage string, operation Operation, namespace Namespace, requirement string, version string, anonymousID string) (*mono_models.Commit, error) {
	changeset := []*mono_models.CommitChangeEditable{
		{
			Operation:         string(operation),
			Namespace:         namespace.String(),
			Requirement:       requirement,
			VersionConstraint: version,
		},
	}

	return AddChangeset(parentCommitID, commitMessage, anonymousID, changeset)
}

func UpdateBranchForProject(pj ProjectInfo, commitID strfmt.UUID) error {
	pjm, err := FetchProjectByName(pj.Owner(), pj.Name())
	if err != nil {
		return errs.Wrap(err, "Could not fetch project")
	}

	branch, err := BranchForProjectByName(pjm, pj.BranchName())
	if err != nil {
		return errs.Wrap(err, "Could not fetch branch: %s", pj.BranchName())
	}

	err = UpdateBranchCommit(branch.BranchID, commitID)
	if err != nil {
		return errs.Wrap(err, "Could no update branch to commit %s", commitID.String())
	}

	return nil
}

// UpdateBranchCommit updates the commit that a branch is pointed at
func UpdateBranchCommit(branchID strfmt.UUID, commitID strfmt.UUID) error {
	changeset := &mono_models.BranchEditable{
		CommitID: &commitID,
	}

	return updateBranch(branchID, changeset)
}

// UpdateBranchTracking updates the tracking information for the given branch
func UpdateBranchTracking(branchID, commitID, trackingBranchID strfmt.UUID, trackingType TrackingType) error {
	tracking := trackingType.String()
	changeset := &mono_models.BranchEditable{
		CommitID:     &commitID,
		TrackingType: &tracking,
		Tracks:       &trackingBranchID,
	}

	return updateBranch(branchID, changeset)
}

func updateBranch(branchID strfmt.UUID, changeset *mono_models.BranchEditable) error {
	params := vcsClient.NewUpdateBranchParams()
	params.SetBranchID(branchID)
	params.SetBranch(changeset)

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
		return locale.NewError("err_update_branch", "", api.ErrorMessageFromPayload(err))
	}
	return nil
}

func DeleteBranch(branchID strfmt.UUID) error {
	params := vcsClient.NewDeleteBranchParams()
	params.SetBranchID(branchID)

	_, err := authentication.Client().VersionControl.DeleteBranch(params, authentication.ClientAuth())
	if err != nil {
		return locale.WrapError(err, "err_delete_branch", "Could not delete branch")
	}

	return nil
}

// CommitPackage commits a package to an existing parent commit
func CommitPackage(parentCommitID strfmt.UUID, operation Operation, packageName, packageNamespace, packageVersion string, anonymousID string) (strfmt.UUID, error) {
	var commitID strfmt.UUID
	languages, err := FetchLanguagesForCommit(parentCommitID)
	if err != nil {
		return commitID, err
	}

	if len(languages) == 0 {
		return commitID, locale.NewError("err_project_no_languages")
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

	namespace := NewNamespacePackage(languages[0].Name)
	if strings.HasPrefix(packageNamespace, NamespaceBundle.Prefix()) {
		namespace = NewNamespaceBundle(languages[0].Name)
	}

	commit, err := AddCommit(
		parentCommitID, locale.Tr(message, packageName, packageVersion),
		operation, namespace,
		packageName, packageVersion, anonymousID,
	)
	if err != nil {
		return commitID, err
	}
	return commit.CommitID, nil
}

// UpdateProjectBranchCommitByName updates the vcs branch for a project given by its namespace with a new commitID
func UpdateProjectBranchCommit(pj ProjectInfo, commitID strfmt.UUID) error {
	pjm, err := FetchProjectByName(pj.Owner(), pj.Name())
	if err != nil {
		return errs.Wrap(err, "Could not fetch project")
	}

	return UpdateProjectBranchCommitWithModel(pjm, pj.BranchName(), commitID)
}

// UpdateProjectBranchCommitByName updates the vcs branch for a project given by its namespace with a new commitID
func UpdateProjectBranchCommitWithModel(pjm *mono_models.Project, branchName string, commitID strfmt.UUID) error {
	branch, err := BranchForProjectByName(pjm, branchName)
	if err != nil {
		return errs.Wrap(err, "Could not fetch branch: %s", branchName)
	}

	err = UpdateBranchCommit(branch.BranchID, commitID)
	if err != nil {
		return errs.Wrap(err, "Could update branch %s to commitID %s", branchName, commitID.String())
	}
	return nil
}

// CommitChangeset commits multiple changes in one commit
func CommitChangeset(parentCommitID strfmt.UUID, commitMsg string, anonymousID string, changeset Changeset) (strfmt.UUID, error) {
	var commitID strfmt.UUID
	languages, err := FetchLanguagesForCommit(parentCommitID)
	if err != nil {
		return commitID, err
	}

	if len(languages) == 0 {
		return commitID, locale.NewError("err_project_no_languages")
	}

	commit, err := AddChangeset(parentCommitID, commitMsg, anonymousID, changeset)
	if err != nil {
		return commitID, err
	}
	return commit.CommitID, nil
}

// CommitInitial creates a root commit for a new branch
func CommitInitial(hostPlatform string, lang *language.Supported, langVersion string) (strfmt.UUID, error) {
	var language string
	if lang != nil {
		language = lang.Requirement()
		if langVersion == "" {
			langVersion = lang.RecommendedVersion()
		}
	}

	platformID, err := hostPlatformToPlatformID(hostPlatform)
	if err != nil {
		return "", err
	}

	var changes []*mono_models.CommitChangeEditable

	if language != "" {
		c := &mono_models.CommitChangeEditable{
			Operation:         string(OperationAdded),
			Namespace:         NewNamespaceLanguage().String(),
			Requirement:       language,
			VersionConstraint: langVersion,
		}
		changes = append(changes, c)
	}

	c := &mono_models.CommitChangeEditable{
		Operation:         string(OperationAdded),
		Namespace:         NewNamespacePlatform().String(),
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
		return "", locale.WrapError(err, "err_add_commit", "", api.ErrorMessageFromPayload(err))
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
func (cs indexedCommits) countBetween(first, last string) (int, error) {
	if first == last {
		return 0, nil
	}

	if last == "" {
		return 0, locale.NewError("err_commit_count_missing_last")
	}

	if first != "" {
		if _, ok := cs[first]; !ok {
			return 0, locale.WrapError(ErrCommitCountUnknowable, "err_commit_count_cannot_find_first", "", first)
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
			return 0, locale.WrapError(ErrCommitCountUnknowable, "err_commit_count_cannot_find", next)
		}
	}

	return ct, nil
}

// CommitPlatform commits a single platform commit
func CommitPlatform(pj ProjectInfo, op Operation, name, version string, word int) error {
	platform, err := FetchPlatformByDetails(name, version, word)
	if err != nil {
		return err
	}

	pjm, err := FetchProjectByName(pj.Owner(), pj.Name())
	if err != nil {
		return errs.Wrap(err, "Could not fetch project")
	}

	branch, err := BranchForProjectByName(pjm, pj.BranchName())
	if err != nil {
		return errs.Wrap(err, "Could not fetch branch: %s", pj.BranchName())
	}

	if branch.CommitID == nil {
		return locale.NewError("err_project_no_languages")
	}

	var msgL10nKey string
	switch op {
	case OperationAdded:
		msgL10nKey = "commit_message_add_platform"
	case OperationUpdated:
		return errs.New("this is not supported yet")
	case OperationRemoved:
		msgL10nKey = "commit_message_removed_platform"
	}

	bCommitID := *branch.CommitID
	msg := locale.Tr(msgL10nKey, name, strconv.Itoa(word), version)
	platformID := platform.PlatformID.String()

	// version is not the value that AddCommit needs - platforms do not post a version
	commit, err := AddCommit(bCommitID, msg, op, NewNamespacePlatform(), platformID, "", "")
	if err != nil {
		return err
	}

	return UpdateBranchCommit(branch.BranchID, commit.CommitID)
}

// CommitLanguage commits a single language to the platform
func CommitLanguage(pj ProjectInfo, op Operation, name, version string) error {
	lang, err := FetchLanguageByDetails(name, version)
	if err != nil {
		return err
	}

	pjm, err := FetchProjectByName(pj.Owner(), pj.Name())
	if err != nil {
		return errs.Wrap(err, "Could not fetch project")
	}

	branch, err := BranchForProjectByName(pjm, pj.BranchName())
	if err != nil {
		return errs.Wrap(err, "Could not fetch branch: %s", pj.BranchName())
	}

	if branch.CommitID == nil {
		return locale.NewError("err_project_no_languages")
	}

	var msgL10nKey string
	switch op {
	case OperationAdded:
		msgL10nKey = "commit_message_add_language"
	case OperationUpdated:
		return errs.New("this is not supported yet")
	case OperationRemoved:
		msgL10nKey = "commit_message_removed_language"
	}

	branchCommitID := *branch.CommitID
	msg := locale.Tr(msgL10nKey, name, version)

	commit, err := AddCommit(branchCommitID, msg, op, NewNamespaceLanguage(), lang.Name, lang.Version, "")
	if err != nil {
		return err
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

func TrackBranch(source, target *mono_models.Project) error {
	sourceBranch, err := DefaultBranchForProject(source)
	if err != nil {
		return err
	}

	targetBranch, err := DefaultBranchForProject(target)
	if err != nil {
		return err
	}

	trackingType := mono_models.BranchEditableTrackingTypeNotify

	updateParams := vcsClient.NewUpdateBranchParams()
	branch := &mono_models.BranchEditable{
		TrackingType: &trackingType,
		Tracks:       &sourceBranch.BranchID,
	}
	updateParams.SetBranch(branch)
	updateParams.SetBranchID(targetBranch.BranchID)

	_, err = authentication.Client().VersionControl.UpdateBranch(updateParams, authentication.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		return locale.WrapError(err, msg)
	}
	return nil
}

func GetRootBranches(branches mono_models.Branches) mono_models.Branches {
	var rootBranches mono_models.Branches
	for _, branch := range branches {
		// Account for forked projects where the root branches contain
		// a tracking ID that is not in the current project's branches
		if branch.Tracks != nil && containsBranch(branch.Tracks, branches) {
			continue
		}
		rootBranches = append(rootBranches, branch)
	}
	return rootBranches
}

func containsBranch(id *strfmt.UUID, branches mono_models.Branches) bool {
	for _, branch := range branches {
		if branch.BranchID.String() == id.String() {
			return true
		}
	}
	return false
}

// GetBranchChildren returns the direct children of the given branch
func GetBranchChildren(branch *mono_models.Branch, branches mono_models.Branches) mono_models.Branches {
	var children mono_models.Branches
	if branch == nil {
		return children
	}

	for _, b := range branches {
		if b.Tracks != nil && b.Tracks.String() == branch.BranchID.String() {
			children = append(children, b)
		}
	}
	return children
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

func RevertCommit(pj ProjectInfo, to strfmt.UUID) error {
	revertCommit, err := GetRevertCommit(pj.CommitUUID(), to)
	if err != nil {
		return err
	}

	addCommit, err := AddRevertCommit(revertCommit)
	if err != nil {
		return err
	}

	pjm, err := FetchProjectByName(pj.Owner(), pj.Name())
	if err != nil {
		return errs.Wrap(err, "Could not fetch project")
	}

	branch, err := BranchForProjectByName(pjm, pj.BranchName())
	if err != nil {
		return errs.Wrap(err, "Could not fetch branch: %s", pj.BranchName())
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
