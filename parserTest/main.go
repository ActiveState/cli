package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/localorder/parser"
	"github.com/ActiveState/cli/pkg/localorder/parser/transform"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

var data = []byte(`
let:
  # This is a comment
  runtime = solve(
    platforms = ["linux", "windows"]
    languages = ["python"]
    packages =  ["requests"]
  )
in: 
  runtime
`)

func main() {
	err := runParser()
	if err != nil {
		fmt.Println(errs.JoinMessage(err))
		os.Exit(1)
	}
}

type MyStruct struct {
	mySlice []int
}

func AppendToSlice(s *MyStruct, newValue int) {
	s.mySlice = append(s.mySlice, newValue)
}

func runParser() error {
	p := parser.New(data)
	t, err := p.Parse()
	if err != nil {
		return errs.Wrap(err, "Failed to parse")
	}

	transformer := transform.NewBuildScriptTransformer(t)
	bs, err := transformer.Transform2()
	if err != nil {
		return errs.Wrap(err, "Failed to transform")
	}

	data, err := json.MarshalIndent(bs, "", "  ")
	if err != nil {
		return errs.Wrap(err, "Failed to marshal")
	}

	fmt.Println(string(data))

	myInstance := MyStruct{mySlice: []int{1, 2, 3}}
	AppendToSlice(&myInstance, 4)
	fmt.Println(myInstance.mySlice) // Output: [1 2 3 4]

	return nil
}

func createGraphVis(root *parser.NodeElement) error {
	g := graphviz.New()
	graph, err := g.Graph()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := graph.Close(); err != nil {
			log.Fatal(err)
		}
		g.Close()
	}()

	addNodes(graph, root, nil)

	var buf bytes.Buffer
	if err := g.Render(graph, graphviz.PNG, &buf); err != nil {
		log.Fatal(err)
	}

	if err := g.RenderFilename(graph, graphviz.PNG, "/Users/mikedrakos/work/cli/graph.png"); err != nil {
		log.Fatal(err)
	}

	return nil
}

func addNodes(graph *cgraph.Graph, t *parser.NodeElement, parent *cgraph.Node) error {
	children := t.Children()

	var err error
	if parent == nil {
		parent, err = graph.CreateNode(t.Type().String())
		if err != nil {
			return errs.Wrap(err, "Failed to create node")
		}
	}

	for _, c := range children {
		node, err := graph.CreateNode(c.Type().String())
		if err != nil {
			return errs.Wrap(err, "Failed to create node")
		}

		label := c.Type().String()
		if c.Type().HasLiteral() {
			label = fmt.Sprintf("%s: %s", c.Type().String(), c.Literal())
		}
		fmt.Println("Label: ", label)
		node.SetLabel(label)
		graph.CreateEdge("", parent, node)
		err = addNodes(graph, c, node)
		if err != nil {
			return errs.Wrap(err, "Failed to add nodes")
		}
	}

	return nil
}
