package branch

import (
	"fmt"
	"sort"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type branchNode struct {
	Label string
	ID    string
}

type tree map[branchNode]tree

type BranchOutput struct {
	branches    mono_models.Branches
	localBranch string
}

const (
	prefixLink string = "│  "
	prefixMid  string = "├─"
	prefixEnd  string = "└─"

	branchFormatting      string = "[NOTICE]%s[/RESET]"
	localBranchFormatting string = "[ACTIONABLE]%s[/RESET] [DISABLED](Current)[/RESET]"
)

func NewBranchOutput(branches mono_models.Branches, localBranch string) *BranchOutput {
	return &BranchOutput{
		branches:    branches,
		localBranch: localBranch,
	}
}

func (bt *BranchOutput) MarshalOutput(format output.Format) interface{} {
	if format != output.PlainFormatName {
		return branchListing(bt.branches, bt.localBranch)
	}
	return branchTree(bt.branches, bt.localBranch)
}

func branchListing(branches mono_models.Branches, localBranch string) []string {
	var branchNames []string
	for _, branch := range branches {
		branchName := applyFormatting(branch.Label, localBranch)
		branchNames = append(branchNames, branchName)
	}
	return branchNames
}

func branchTree(branches mono_models.Branches, localBranch string) string {
	tree := make(tree)
	for _, branch := range model.GetRootBranches(branches) {
		tree[branchNode{branch.Label, branch.BranchID.String()}] = buildBranchTree(branch, branches)
	}

	var levelsCompleted []int
	rootBranches := model.GetRootBranches(branches)
	return treeString(tree, rootBranches, localBranch, 0, levelsCompleted)
}

func buildBranchTree(currentBranch *mono_models.Branch, branches mono_models.Branches) tree {
	children := getChildren(currentBranch, branches)
	for _, branch := range branches {
		// Discard any branches without tracking information as we are only interested
		// in child branches of the current branch
		if branch.Tracks == nil {
			continue
		}

		// Check that this branch is a child branch and recursively build its tree
		node := branchNode{branch.Label, branch.BranchID.String()}
		if _, ok := children[node]; ok {
			children[node] = buildBranchTree(branch, branches)
		}
	}
	return children
}

func getChildren(branch *mono_models.Branch, branches mono_models.Branches) tree {
	childTree := make(tree)
	children := model.GetBranchChildren(branch, branches)

	for _, child := range children {
		childTree[branchNode{child.Label, child.BranchID.String()}] = make(tree)
	}
	return childTree
}

func treeString(currentTree tree, rootBranches mono_models.Branches, localBranch string, currentLevel int, levelsCompleted []int) string {
	// Sort keys at current level to ensure consistent output
	var nodes []branchNode
	for k := range currentTree {
		nodes = append(nodes, k)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Label < nodes[j].Label
	})

	var output string
	for i, node := range nodes {
		prefix := prefixMid
		if i == len(nodes)-1 {
			levelsCompleted = append(levelsCompleted, currentLevel)
			prefix = prefixEnd
		}
		if isRootNode(node, rootBranches) {
			prefix = ""
		}

		output += nodeString(node, rootBranches, localBranch, currentLevel, levelsCompleted, prefix)
		if len(currentTree[node]) > 0 {
			output += treeString(currentTree[node], rootBranches, localBranch, currentLevel+1, levelsCompleted)
		}
	}

	return output
}

func nodeString(node branchNode, rootBranches mono_models.Branches, localBranch string, currentLevel int, levelsCompleted []int, prefix string) string {
	// Print necessary prefix links for current depth
	var output string
	for i := 0; i < currentLevel; i++ {
		// Apply spacing for completed levels, print prefix link for incomplete levels
		// Do not print prefix links for projects with multiple root-level branches
		if isCompleted(levelsCompleted, i) || (i == 0 && len(rootBranches) > 1) {
			output += " "
			continue
		}
		output += prefixLink
	}

	branchName := applyFormatting(node.Label, localBranch)
	format := "%s %s\n"
	if prefix == "" {
		format = "%s%s\n"
	}

	output += fmt.Sprintf(format, prefix, branchName)
	return output
}

func applyFormatting(label, localBranch string) string {
	branchName := fmt.Sprintf(branchFormatting, label)
	if label == localBranch {
		branchName = fmt.Sprintf(localBranchFormatting, label)
	}
	return branchName
}

func isRootNode(node branchNode, rootBranches mono_models.Branches) bool {
	for _, branch := range rootBranches {
		if branch.Label == node.Label {
			return true
		}
	}
	return false
}

func isCompleted(levelsCompleted []int, level int) bool {
	for _, l := range levelsCompleted {
		if l == level {
			return true
		}
	}
	return false
}
