package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	vcsClient "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/go-openapi/strfmt"
)

var (
	ErrCommitCountUnknowable = errs.New("Commit count is unknowable")

	ErrMergeFastForward = errs.New("No merge required")

	ErrMergeCommitInHistory = errs.New("Can't merge commit thats already in target commits history")

	ErrCommitNotInHistory = errs.New("Target commit is not in history")
)

var ErrOrderForbidden = errs.New("no permission to retrieve order")

type ErrUpdateBranchAuth struct{ *locale.LocalizedError }

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
	NamespacePackageMatch = `^language\/(\w+)$`

	// NamespacePackageMatch is the namespace used for language package requirements
	NamespaceBuilderMatch = `^builder(-lib){0,1}$`

	// NamespaceBundlesMatch is the namespace used for bundle package requirements
	NamespaceBundlesMatch = `^bundles\/(\w+)$`

	// NamespacePrePlatformMatch is the namespace used for pre-platform bits
	NamespacePrePlatformMatch = `^pre-platform-installer$`

	// NamespaceCamelFlagsMatch is the namespace used for passing camel flags
	NamespaceCamelFlagsMatch = `^camel-flags$`

	// NamespaceOrgMatch is the namespace used for org specific requirements
	NamespaceOrgMatch = `^private\/`

	// NamespaceBuildFlagsMatch is the namespace used for passing build flags
	NamespaceBuildFlagsMatch = `^build-flags$`
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
		multilog.Error("Could not match regex for %v, query: %s, error: %v", namespace, query, err)
	}
	return match
}

type NamespaceType struct {
	name      string
	prefix    string
	matchable NamespaceMatchable
}

var (
	NamespacePackage  = NamespaceType{"package", "language", NamespacePackageMatch} // these values should match the namespace prefix
	NamespaceBundle   = NamespaceType{"bundle", "bundles", NamespaceBundlesMatch}
	NamespaceLanguage = NamespaceType{"language", "", NamespaceLanguageMatch}
	NamespacePlatform = NamespaceType{"platform", "", NamespacePlatformMatch}
	NamespaceOrg      = NamespaceType{"org", "private/", NamespaceOrgMatch}
	NamespaceRaw      = NamespaceType{"raw", "", ""}
	NamespaceBlank    = NamespaceType{"", "", ""}
)

func (t NamespaceType) String() string {
	return t.name
}

func (t NamespaceType) Prefix() string {
	return t.prefix
}

func (t NamespaceType) Matchable() NamespaceMatchable {
	return t.matchable
}

// Namespace is the type used for communicating namespaces, mainly just allows for self documenting code
type Namespace struct {
	nsType NamespaceType
	value  string
}

func (n Namespace) IsValid() bool {
	return n.nsType.name != "" && n.nsType != NamespaceBlank && n.value != ""
}

func (n Namespace) Type() NamespaceType {
	return n.nsType
}

func (n Namespace) String() string {
	return n.value
}

func ParseNamespace(ns string) Namespace {
	if ns == "" {
		return Namespace{NamespaceBlank, ns}
	}
	for _, n := range []NamespaceType{NamespacePackage, NamespaceBundle, NamespaceLanguage, NamespacePlatform, NamespaceOrg} {
		if NamespaceMatch(ns, n.Matchable()) {
			return Namespace{n, ns}
		}
	}
	return Namespace{nsType: NamespaceRaw, value: ns}
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

func NewNamespaceRaw(value string) Namespace {
	return Namespace{NamespaceRaw, value}
}

func NewBlankNamespace() Namespace {
	return Namespace{NamespaceBlank, ""}
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

func NewOrgNamespace(orgName string) Namespace {
	return Namespace{
		nsType: NamespaceOrg,
		value:  NamespaceOrg.prefix + "/" + orgName,
	}
}

func LanguageFromNamespace(ns string) string {
	matchables := []NamespaceMatchable{
		NamespacePackage.Matchable(),
		NamespaceBundle.Matchable(),
	}
	for _, m := range matchables {
		if NamespaceMatch(ns, m) {
			return strings.Split(ns, "/")[1]
		}
	}
	return ""
}

// FilterSupportedIngredients filters a list of ingredients, returning only those that are currently supported (such that they can be built) by the Platform
func FilterSupportedIngredients(supported []model.SupportedLanguage, ingredients []*IngredientAndVersion) ([]*IngredientAndVersion, error) {
	var res []*IngredientAndVersion

	for _, i := range ingredients {
		language := LanguageFromNamespace(*i.Ingredient.PrimaryNamespace)

		for _, l := range supported {
			if l.Name != language {
				continue
			}
			res = append(res, i)
			break
		}
	}

	return res, nil
}

// BranchCommitID returns the latest commit id by owner and project names. It
// is possible for a nil commit id to be returned without failure.
func BranchCommitID(ownerName, projectName, branchName string) (*strfmt.UUID, error) {
	proj, err := LegacyFetchProjectByName(ownerName, projectName)
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
			"Your project does not have any commits yet. Head over to {{.V0}} to set up your project.", api.GetPlatformURL(fmt.Sprintf("%s/%s", ownerName, projectName)).String())
	}

	return branch.CommitID, nil
}

func CommitBelongsToBranch(ownerName, projectName, branchName string, commitID strfmt.UUID, auth *authentication.Auth) (bool, error) {
	latestCID, err := BranchCommitID(ownerName, projectName, branchName)
	if err != nil {
		return false, errs.Wrap(err, "Could not get latest commit ID of branch")
	}

	return CommitWithinCommitHistory(*latestCID, commitID, auth)
}

func CommitWithinCommitHistory(latestCommitID, searchCommitID strfmt.UUID, auth *authentication.Auth) (bool, error) {
	history, err := CommitHistoryFromID(latestCommitID, auth)
	if err != nil {
		return false, errs.Wrap(err, "Could not get commit history from commit ID")
	}

	for _, commit := range history {
		if commit.CommitID == searchCommitID {
			return true, nil
		}
	}

	return false, nil
}

// CommitHistory will return the commit history for the given owner / project
func CommitHistory(ownerName, projectName, branchName string, auth *authentication.Auth) ([]*mono_models.Commit, error) {
	latestCID, err := BranchCommitID(ownerName, projectName, branchName)
	if err != nil {
		return nil, err
	}
	return commitHistory(*latestCID, auth)
}

// CommitHistoryFromID will return the commit history from the given commitID
func CommitHistoryFromID(commitID strfmt.UUID, auth *authentication.Auth) ([]*mono_models.Commit, error) {
	return commitHistory(commitID, auth)
}

func commitHistory(commitID strfmt.UUID, auth *authentication.Auth) ([]*mono_models.Commit, error) {
	offset := int64(0)
	limit := int64(100)
	var commits []*mono_models.Commit

	cont := true
	for cont {
		payload, err := CommitHistoryPaged(commitID, offset, limit, auth)
		if err != nil {
			return commits, err
		}
		commits = append(commits, payload.Commits...)
		offset += limit
		cont = payload.TotalCommits > offset
	}

	return commits, nil
}

// CommitHistoryPaged will return the commit history for the given owner / project
func CommitHistoryPaged(commitID strfmt.UUID, offset, limit int64, auth *authentication.Auth) (*mono_models.CommitHistoryInfo, error) {
	params := vcsClient.NewGetCommitHistoryParams()
	params.SetCommitID(commitID)
	params.Limit = &limit
	params.Offset = &offset

	var res *vcsClient.GetCommitHistoryOK
	var err error
	if auth.Authenticated() {
		authClient, err := auth.Client()
		if err != nil {
			return nil, errs.Wrap(err, "Could not get auth client")
		}
		res, err = authClient.VersionControl.GetCommitHistory(params, auth.ClientAuth())
		if err != nil {
			return nil, locale.WrapError(err, "err_get_commit_history", "", api.ErrorMessageFromPayload(err))
		}
	} else {
		res, err = mono.New().VersionControl.GetCommitHistory(params, nil)
		if err != nil {
			return nil, locale.WrapError(err, "err_get_commit_history", "", api.ErrorMessageFromPayload(err))
		}
	}

	return res.Payload, nil
}

// CommonParent returns the first commit id which both provided commit id
// histories have in common.
func CommonParent(commit1, commit2 *strfmt.UUID, auth *authentication.Auth) (*strfmt.UUID, error) {
	if commit1 == nil || commit2 == nil {
		return nil, nil
	}

	if *commit1 == *commit2 {
		return commit1, nil
	}

	history1, err := CommitHistoryFromID(*commit1, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get commit history for %s", commit1.String())
	}

	history2, err := CommitHistoryFromID(*commit2, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get commit history for %s", commit2.String())
	}

	return commonParentWithHistory(commit1, commit2, history1, history2), nil
}

func commonParentWithHistory(commit1, commit2 *strfmt.UUID, history1, history2 []*mono_models.Commit) *strfmt.UUID {
	if commit1 == nil || commit2 == nil {
		return nil
	}

	if *commit1 == *commit2 {
		return commit1
	}

	for _, c := range history1 {
		if c.CommitID == *commit2 {
			return commit2 // commit1 history contains commit2
		}
		for _, c2 := range history2 {
			if c.CommitID == c2.CommitID {
				return &c.CommitID // commit1 and commit2 have a common parent
			}
		}
	}

	for _, c2 := range history2 {
		if c2.CommitID == *commit1 {
			return commit1 // commit2 history contains commit1
		}
	}

	return nil
}

// CommitsBehind compares the provided commit id with the latest commit
// id and returns the count of commits it is behind. A negative return value
// indicates the provided commit id is ahead of the latest commit id (that is,
// there are local commits).
func CommitsBehind(latestCID, currentCommitID strfmt.UUID, auth *authentication.Auth) (int, error) {
	if latestCID == "" {
		if currentCommitID == "" {
			return 0, nil // ok, nothing to do
		}
		return 0, locale.NewError("err_commit_count_no_latest_with_commit")
	}

	if latestCID.String() == currentCommitID.String() {
		return 0, nil
	}

	// Assume current is behind or equal to latest.
	commits, err := CommitHistoryFromID(latestCID, auth)
	if err != nil {
		return 0, locale.WrapError(err, "err_get_commit_history", "", err.Error())
	}

	indexed := makeIndexedCommits(commits)
	if behind, err := indexed.countBetween(currentCommitID.String(), latestCID.String()); err == nil {
		return behind, nil
	}

	// Assume current is ahead of latest.
	commits, err = CommitHistoryFromID(currentCommitID, auth)
	if err != nil {
		return 0, locale.WrapError(err, "err_get_commit_history", "", err.Error())
	}

	indexed = makeIndexedCommits(commits)
	ahead, err := indexed.countBetween(latestCID.String(), currentCommitID.String())
	return -ahead, err
}

// Changeset aliases for eased usage and to act as a disconnect from the underlying dep.
type Changeset = []*mono_models.CommitChangeEditable

func UpdateBranchForProject(pj ProjectInfo, commitID strfmt.UUID, auth *authentication.Auth) error {
	pjm, err := LegacyFetchProjectByName(pj.Owner(), pj.Name())
	if err != nil {
		return errs.Wrap(err, "Could not fetch project")
	}

	branch, err := BranchForProjectByName(pjm, pj.BranchName())
	if err != nil {
		return errs.Wrap(err, "Could not fetch branch: %s", pj.BranchName())
	}

	err = UpdateBranchCommit(branch.BranchID, commitID, auth)
	if err != nil {
		return errs.Wrap(err, "Could no update branch to commit %s", commitID.String())
	}

	return nil
}

// UpdateBranchCommit updates the commit that a branch is pointed at
func UpdateBranchCommit(branchID strfmt.UUID, commitID strfmt.UUID, auth *authentication.Auth) error {
	changeset := &mono_models.BranchEditable{
		CommitID: &commitID,
	}

	return updateBranch(branchID, changeset, auth)
}

// UpdateBranchTracking updates the tracking information for the given branch
func UpdateBranchTracking(branchID, commitID, trackingBranchID strfmt.UUID, trackingType TrackingType, auth *authentication.Auth) error {
	tracking := trackingType.String()
	changeset := &mono_models.BranchEditable{
		CommitID:     &commitID,
		TrackingType: &tracking,
		Tracks:       &trackingBranchID,
	}

	return updateBranch(branchID, changeset, auth)
}

func updateBranch(branchID strfmt.UUID, changeset *mono_models.BranchEditable, auth *authentication.Auth) error {
	authClient, err := auth.Client()
	if err != nil {
		return errs.Wrap(err, "Could not get auth client")
	}

	params := vcsClient.NewUpdateBranchParams()
	params.SetBranchID(branchID)
	params.SetBranch(changeset)

	_, err = authClient.VersionControl.UpdateBranch(params, auth.ClientAuth())
	if err != nil {
		if _, ok := err.(*version_control.UpdateBranchForbidden); ok {
			return &ErrUpdateBranchAuth{locale.NewExternalError("err_branch_update_auth", "Branch update failed with authentication error")}
		}
		return locale.NewError("err_update_branch", "", api.ErrorMessageFromPayload(err))
	}
	return nil
}

func DeleteBranch(branchID strfmt.UUID, auth *authentication.Auth) error {
	authClient, err := auth.Client()
	if err != nil {
		return errs.Wrap(err, "Could not get auth client")
	}

	params := vcsClient.NewDeleteBranchParams()
	params.SetBranchID(branchID)

	_, err = authClient.VersionControl.DeleteBranch(params, auth.ClientAuth())
	if err != nil {
		return locale.WrapError(err, "err_delete_branch", "Could not delete branch")
	}

	return nil
}

// UpdateProjectBranchCommitByName updates the vcs branch for a project given by its namespace with a new commitID
func UpdateProjectBranchCommit(pj ProjectInfo, commitID strfmt.UUID, auth *authentication.Auth) error {
	pjm, err := LegacyFetchProjectByName(pj.Owner(), pj.Name())
	if err != nil {
		return errs.Wrap(err, "Could not fetch project")
	}

	return UpdateProjectBranchCommitWithModel(pjm, pj.BranchName(), commitID, auth)
}

// UpdateProjectBranchCommitByName updates the vcs branch for a project given by its namespace with a new commitID
func UpdateProjectBranchCommitWithModel(pjm *mono_models.Project, branchName string, commitID strfmt.UUID, auth *authentication.Auth) error {
	branch, err := BranchForProjectByName(pjm, branchName)
	if err != nil {
		return errs.Wrap(err, "Could not fetch branch: %s", branchName)
	}

	err = UpdateBranchCommit(branch.BranchID, commitID, auth)
	if err != nil {
		return errs.Wrap(err, "Could update branch %s to commitID %s", branchName, commitID.String())
	}
	return nil
}

// CommitInitial creates a root commit for a new branch
func CommitInitial(hostPlatform string, langName, langVersion string, auth *authentication.Auth) (strfmt.UUID, error) {
	platformID, err := PlatformNameToPlatformID(hostPlatform)
	if err != nil {
		return "", err
	}

	var changes []*mono_models.CommitChangeEditable

	if langName != "" {
		versionConstraints, err := versionStringToConstraints(langVersion)
		if err != nil {
			return "", errs.Wrap(err, "Could not process version string into constraints")
		}
		c := &mono_models.CommitChangeEditable{
			Operation:          string(OperationAdded),
			Namespace:          NewNamespaceLanguage().String(),
			Requirement:        langName,
			VersionConstraints: versionConstraints,
		}
		changes = append(changes, c)
	}

	c := &mono_models.CommitChangeEditable{
		Operation:   string(OperationAdded),
		Namespace:   NewNamespacePlatform().String(),
		Requirement: platformID,
	}
	changes = append(changes, c)

	commit := &mono_models.CommitEditable{
		Changeset: changes,
		Message:   locale.T("commit_message_add_initial"),
	}
	params := vcsClient.NewAddCommitParams()
	params.SetCommit(commit)

	res, err := mono.New().VersionControl.AddCommit(params, auth.ClientAuth())
	if err != nil {
		return "", locale.WrapError(err, "err_add_commit", "", api.ErrorMessageFromPayload(err))
	}

	return res.Payload.CommitID, nil
}

func versionStringToConstraints(version string) ([]*mono_models.Constraint, error) {
	requirements, err := bpModel.VersionStringToRequirements(version)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to process version string into requirements")
	}

	constraints := make([]*mono_models.Constraint, len(requirements))
	for i, constraint := range requirements {
		constraints[i] = &mono_models.Constraint{
			Comparator: constraint[types.VersionRequirementComparatorKey],
			Version:    constraint[types.VersionRequirementVersionKey],
		}
	}
	return constraints, nil
}

type indexedCommits map[string]string // key == commit id / val == parent id

func makeIndexedCommits(cs []*mono_models.Commit) indexedCommits {
	m := make(indexedCommits)

	for _, c := range cs {
		m[string(c.CommitID)] = string(c.ParentCommitID)
	}

	return m
}

// countBetween returns 0 if same or if unable to determine the count.
// Caution: Currently, the logic does not verify that the first commit is "before" the last commit.
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

func ResolveRequirementNameAndVersion(name, version string, word int, namespace Namespace, auth *authentication.Auth) (string, string, error) {
	if namespace.Type() == NamespacePlatform {
		platform, err := FetchPlatformByDetails(name, version, word)
		if err != nil {
			return "", "", errs.Wrap(err, "Could not fetch platform")
		}
		name = platform.PlatformID.String()
		version = ""
	}

	return name, version, nil
}

func ChangesetFromRequirements(op Operation, reqs []*gqlModel.Requirement) Changeset {
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

func TrackBranch(source, target *mono_models.Project, auth *authentication.Auth) error {
	authClient, err := auth.Client()
	if err != nil {
		return errs.Wrap(err, "Could not get auth client")
	}

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

	_, err = authClient.VersionControl.UpdateBranch(updateParams, auth.ClientAuth())
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

func GetRevertCommit(from, to strfmt.UUID, auth *authentication.Auth) (*mono_models.Commit, error) {
	params := vcsClient.NewGetRevertCommitParams()
	params.SetCommitFromID(from)
	params.SetCommitToID(to)

	client := mono.New()
	var err error
	if auth.Authenticated() {
		client, err = auth.Client()
		if err != nil {
			return nil, errs.Wrap(err, "Could not get auth client")
		}
	}
	res, err := client.VersionControl.GetRevertCommit(params, auth.ClientAuth())
	if err != nil {
		return nil, locale.WrapError(err, "err_get_revert_commit", "Could not revert from commit ID {{.V0}} to {{.V1}}", from.String(), to.String())
	}

	return res.Payload, nil
}

func RevertCommitWithinHistory(from, to, latest strfmt.UUID, auth *authentication.Auth) (*mono_models.Commit, error) {
	targetCommit := from
	preposition := ""
	if from == latest { // reverting to
		targetCommit = to
		preposition = " to" // need leading whitespace
	}
	ok, err := CommitWithinCommitHistory(latest, targetCommit, auth)
	if err != nil {
		return nil, errs.Wrap(err, "API communication failed.")
	}
	if !ok {
		return nil, locale.WrapError(err, "err_revert_commit_within_history_not_in", "The commit being reverted{{.V0}} is not within the current commit's history.", preposition)
	}

	return RevertCommit(from, to, latest, auth)
}

func RevertCommit(from, to, latest strfmt.UUID, auth *authentication.Auth) (*mono_models.Commit, error) {
	revertCommit, err := GetRevertCommit(from, to, auth)
	if err != nil {
		return nil, err
	}
	if from != latest {
		// The platform assumes revert commits are reverting to a particular commit, rather than
		// reverting the changes in a commit. As a result, commit messages are of the form "Revert to
		// commit X" and parent commit IDs are X. Change the message to reflect the fact we're
		// reverting changes from X and change the parent to be the latest commit so that the revert
		// commit applies to the latest project commit.
		revertCommit.Message = locale.Tl("revert_commit", "Revert commit {{.V0}}", from.String())
		revertCommit.ParentCommitID = latest
	}

	addCommit, err := AddRevertCommit(revertCommit, auth)
	if err != nil {
		return nil, err
	}
	return addCommit, nil
}

func MergeCommit(commitReceiving, commitWithChanges strfmt.UUID) (*mono_models.MergeStrategies, error) {
	params := vcsClient.NewMergeCommitsParams()
	params.SetCommitReceivingChanges(commitReceiving)
	params.SetCommitWithChanges(commitWithChanges)
	params.SetHTTPClient(api.NewHTTPClient())

	res, noContent, err := mono.New().VersionControl.MergeCommits(params)
	if err != nil {
		if api.ErrorCodeFromPayload(err) == 409 {
			logging.Debug("Received 409 from MergeCommit: %s", err.Error())
			return nil, ErrMergeCommitInHistory
		}
		return nil, locale.WrapError(err, "err_api_mergecommit", api.ErrorMessageFromPayload(err))
	}
	if noContent != nil {
		return nil, ErrMergeFastForward
	}

	return res.Payload, nil
}

func MergeRequired(commitReceiving, commitWithChanges strfmt.UUID) (bool, error) {
	_, err := MergeCommit(commitReceiving, commitWithChanges)
	if err != nil {
		if errors.Is(err, ErrMergeFastForward) || errors.Is(err, ErrMergeCommitInHistory) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func GetCommit(commitID strfmt.UUID, auth *authentication.Auth) (*mono_models.Commit, error) {
	params := vcsClient.NewGetCommitParams()
	params.SetCommitID(commitID)
	params.SetHTTPClient(api.NewHTTPClient())

	client := mono.New()
	var err error
	if auth.Authenticated() {
		client, err = auth.Client()
		if err != nil {
			return nil, errs.Wrap(err, "Could not get auth client")
		}
	}
	res, err := client.VersionControl.GetCommit(params, auth.ClientAuth())
	if err != nil {
		return nil, locale.WrapError(err, "err_get_commit", "Could not get commit from ID: {{.V0}}", commitID.String())
	}
	return res.Payload, nil
}

func GetCommitWithinCommitHistory(currentCommitID, targetCommitID strfmt.UUID, auth *authentication.Auth) (*mono_models.Commit, error) {
	commit, err := GetCommit(targetCommitID, auth)
	if err != nil {
		return nil, err
	}

	ok, err := CommitWithinCommitHistory(currentCommitID, targetCommitID, auth)
	if err != nil {
		return nil, errs.Wrap(err, "API communication failed.")
	}
	if !ok {
		return nil, ErrCommitNotInHistory
	}

	return commit, nil
}

// GetCommitWithinProjectHistory searches for the a commit with the given commit ID in the given
// project (including all of its branch history) and returns it if found. Otherwise, it returns
// ErrCommitNotInHistory.
// It doesn't matter if the commit exists in multiple branches, as commits do not belong to
// branches.
// This function exists primarily as an existence check because the buildplanner API currently
// accepts a query for a org/project#commitID even if commitID does not belong to org/project.
// See DS-1705 (yes, DS, not DX).
func GetCommitWithinProjectHistory(commitID strfmt.UUID, owner, name string, localCommitID *strfmt.UUID, auth *authentication.Auth) (*mono_models.Commit, error) {
	commit, err := GetCommit(commitID, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get commit")
	}

	if localCommitID != nil {
		if ok, err := CommitWithinCommitHistory(*localCommitID, commitID, auth); err == nil && ok {
			return commit, nil
		} else if err != nil {
			return nil, errs.Wrap(err, "Unable to determine if commit exists in local history")
		}
	}

	branches, err := BranchesForProject(owner, name)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get branches for project")
	}
	for _, branch := range branches {
		ok, err := CommitWithinCommitHistory(*branch.CommitID, commitID, auth)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to determine if commit exists in branch history")
		}
		if ok {
			return commit, nil
		}
	}

	return nil, ErrCommitNotInHistory
}

func AddRevertCommit(commit *mono_models.Commit, auth *authentication.Auth) (*mono_models.Commit, error) {
	params := vcsClient.NewAddCommitParams()

	editableCommit, err := commitToCommitEditable(commit)
	if err != nil {
		return nil, locale.WrapError(err, "err_convert_commit", "Could not convert commit data")
	}
	params.SetCommit(editableCommit)

	res, err := mono.New().VersionControl.AddCommit(params, auth.ClientAuth())
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
