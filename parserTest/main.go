package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/localorder/parser"
	"github.com/ActiveState/cli/pkg/localorder/parser/transform"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

var data = []byte(`
let:
  # TODO: Add support for solve_legacy function identifier
  runtime = solve(
    platforms = ["linux", "windows"]
    languages = ["python"]
    requirements = ["requests"]
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

func runParser() error {
	cwd, err := environment.GetRootPath()
	if err != nil {
		return errs.Wrap(err, "Failed to get root path")
	}

	// testMapBuildScript()
	// testMapBuildScriptUnmarshal()

	testData, err := os.ReadFile(filepath.Join(cwd, "pkg", "localorder", "parser", "testdata", "moderate.lo"))
	if err != nil {
		return errs.Wrap(err, "Failed to read file")
	}

	p := parser.New(testData)
	t, err := p.Parse()
	if err != nil {
		return errs.Wrap(err, "Failed to parse")
	}

	// err = createGraphVis(t.Root)
	// if err != nil {
	// 	return errs.Wrap(err, "Failed to create graph")
	// }

	transformer := transform.NewBuildScriptTransformer(t)
	bs, err := transformer.Transform()
	if err != nil {
		return errs.Wrap(err, "Failed to transform")
	}

	data, err := json.MarshalIndent(bs, "", "  ")
	if err != nil {
		return errs.Wrap(err, "Failed to marshal")
	}

	fmt.Println(string(data))

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

func testMapBuildScript() error {
	result := map[string]interface{}{
		"let": map[string]interface{}{
			"runtime": map[string]interface{}{
				"solve": map[string]interface{}{
					"platforms": []interface{}{"linux", "windows"},
					"languages": []interface{}{"python"},
					"requirements": []interface{}{
						map[string]interface{}{
							"requests": map[string]interface{}{
								"version": "2.25.1",
							},
						},
					},
				},
			},
		},
		"in": map[string]interface{}{
			"runtime": map[string]interface{}{},
		},
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println("Map build script:", string(data))

	return nil
}

func testMapBuildScriptUnmarshal() error {
	data := []byte(`
	{
		"let": {
			"runtime": {
				"solve": {
					"platforms": ["linux", "windows"],
					"languages": ["python"],
					"requirements": [
						{
							"requests": {
								"version": "2.25.1"
							}
						}
					]
				}
			}
		},
		"in": {
			"runtime": {}
		}
	}
	`)

	var result transform.BuildScript
	err := json.Unmarshal(data, &result)
	if err != nil {
		return err
	}

	fmt.Println("Map build script unmarshal:", result)

	return nil
}
