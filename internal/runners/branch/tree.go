package branch

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

type BranchNode struct {
	Label string
	ID    string
}

type tree map[BranchNode]tree

type BranchTree struct {
	tree                  tree
	branches              mono_models.Branches
	rootBranches          mono_models.Branches
	multipleRoots         bool
	localBranch           string
	branchFormatting      string
	localBranchFormatting string
}

const (
	edgeLink string = "│  "
	edgeMid  string = "├─"
	edgeEnd  string = "└─"
)

func NewBranchTree() *BranchTree {
	return &BranchTree{tree: make(tree)}
}

func (bt *BranchTree) BuildFromBranches(branches mono_models.Branches) {
	bt.rootBranches = getRootBranches(branches)
	if len(bt.rootBranches) > 1 {
		bt.multipleRoots = true
	}
	for _, branch := range bt.rootBranches {
		bt.tree[BranchNode{branch.Label, branch.BranchID.String()}] = buildBranchTree(branch, branches)
	}
}

func buildBranchTree(currentBranch *mono_models.Branch, branches mono_models.Branches) tree {
	t := getChildren(currentBranch, branches)
	for _, branch := range branches {
		if branch.Tracks == nil {
			continue
		}
		if _, ok := t[BranchNode{branch.Label, branch.BranchID.String()}]; ok {
			t[BranchNode{branch.Label, branch.BranchID.String()}] = buildBranchTree(branch, branches)
		}
	}
	return t
}

func getRootBranches(branches mono_models.Branches) mono_models.Branches {
	var rootBranches mono_models.Branches
	for _, branch := range branches {
		if branch.Tracks != nil {
			continue
		}
		rootBranches = append(rootBranches, branch)
	}
	return rootBranches
}

func getChildren(branch *mono_models.Branch, branches mono_models.Branches) tree {
	children := make(tree)
	if branch == nil {
		return children
	}

	for _, b := range branches {
		if b.Tracks == nil {
			continue
		}
		if b.Tracks.String() == branch.BranchID.String() {
			children[BranchNode{b.Label, b.BranchID.String()}] = make(tree)
		}
	}
	return children
}

func (bt *BranchTree) AddBranchFormatting(formatting string) {
	bt.branchFormatting = formatting
}

func (bt *BranchTree) AddLocalBranch(branch string) {
	bt.localBranch = branch
}

func (bt *BranchTree) AddLocalBranchFormatting(formatting string) {
	bt.localBranchFormatting = formatting
}

func (bt *BranchTree) String() string {
	w := new(bytes.Buffer)
	var levelsCompleted []int
	bt.print(w, bt.tree, 0, levelsCompleted)
	return w.String()
}

func (bt *BranchTree) print(w io.Writer, currentTree tree, currentLevel int, levelsCompleted []int) {
	// Sort current keys to ensure consistent output
	var nodes []BranchNode
	for k := range currentTree {
		nodes = append(nodes, k)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Label < nodes[j].Label
	})

	for i, node := range nodes {
		edge := edgeMid
		if i == len(nodes)-1 {
			levelsCompleted = append(levelsCompleted, currentLevel)
			edge = edgeEnd
		}
		if bt.isRootNode(node) {
			edge = ""
		}

		bt.printNode(w, node, currentLevel, levelsCompleted, edge)
		if len(currentTree[node]) > 0 {
			bt.print(w, currentTree[node], currentLevel+1, levelsCompleted)
		}
	}
}

func (bt *BranchTree) printNode(w io.Writer, node BranchNode, currentLevel int, levelsCompleted []int, edge string) {
	// Print necessary edge links for current depth
	for i := 0; i < currentLevel; i++ {
		// Apply spacing for completed levels
		// Do not print edge links for projects with multiple root nodes
		if isCompleted(levelsCompleted, i) || (i == 0 && len(bt.rootBranches) > 1) {
			fmt.Fprintf(w, "%s", strings.Repeat(" ", 2))
			continue
		}
		fmt.Fprintf(w, "%s", edgeLink)
	}

	// Apply formatting
	branchName := fmt.Sprintf(bt.branchFormatting, node.Label)
	if node.Label == bt.localBranch {
		branchName = fmt.Sprintf(bt.localBranchFormatting, node.Label)
	}

	fmt.Fprintf(w, "%s %s\n", edge, branchName)
}

func (bt *BranchTree) isRootNode(node BranchNode) bool {
	for _, branch := range bt.rootBranches {
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
