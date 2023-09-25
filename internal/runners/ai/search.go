package ai

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"golang.org/x/net/context"
)

func init() {
	configMediator.RegisterOption(constants.OpenAITokenConfig, configMediator.String, configMediator.EmptyEvent, configMediator.EmptyEvent)
}

type primeable interface {
	primer.Outputer
	primer.Configurer
}

type Params struct {
	Query    string
	GptModel string
}

type Search struct {
	out output.Outputer
	cfg *config.Instance
}

func New(p primeable) *Search {
	return &Search{
		out: p.Output(),
		cfg: p.Config(),
	}
}

func rationalizeError(err *error) {
	return
}

type ErrNoToken struct{ error }

func (s *Search) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	v, err := NewView(params.Query)
	if err != nil {
		return err
	}

	p := tea.NewProgram(v, tea.WithAltScreen())

	go func() {
		if params.Query == "mock" {
			v.packages = mockedPackages()
			return
		}
		packages, err := s.Search(params.GptModel, params.Query)
		if err != nil {
			panic(errs.JoinMessage(err))
		}
		if len(packages) > 0 {
			v.packages = packages
		}
	}()

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil

	return nil
}

type ChatResult struct {
	Packages []*Package `json:"packages"`
}

type Package struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Example       string   `json:"example"`
	Advantages    []string `json:"advantages"`
	Disadvantages []string `json:"disadvantages"`
	Projects      []string `json:"projects"`
	Rank          int      `json:"rank"`
}

func (s *Search) Search(model, query string) ([]*Package, error) {
	token := s.cfg.GetString(constants.OpenAITokenConfig)
	if token == "" {
		return nil, &ErrNoToken{errs.New("token not set")}
	}

	if model == "" {
		model = openai.GPT4
	}

	client := openai.NewClient(token)
	req := openai.ChatCompletionRequest{
		Temperature: 0,
		Model:       model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "The user is searching for Python packages. " +
					"Narrow your search down to Python packages listed on pypi.org. " +
					"Always produce 5 results.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: query,
			},
		},
		Functions: []openai.FunctionDefinition{
			{
				"PackageResults",
				"Python packages that match the search query.",
				jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"packages": {
							Type:        jsonschema.Array,
							Description: "The actual python packages",
							Items: &jsonschema.Definition{
								Type: jsonschema.Object,
								Properties: map[string]jsonschema.Definition{
									"name": {
										Type:        jsonschema.String,
										Description: "Name of the package",
									},
									"description": {
										Type:        jsonschema.String,
										Description: "Description of the package. Assume the user already knows that this is a package.",
									},
									"example": {
										Type:        jsonschema.String,
										Description: "highly detailed example on how the package is used",
									},
									"advantages": {
										Type:        jsonschema.Array,
										Description: "a list of multiple (more than one) advantages of this package over alternatives",
										Items: &jsonschema.Definition{
											Type: "string",
										},
									},
									"disadvantages": {
										Type:        jsonschema.Array,
										Description: "a list of multiple (more than one) disadvantages of this package over alternatives",
										Items: &jsonschema.Definition{
											Type: "string",
										},
									},
									"projects": {
										Type:        jsonschema.Array,
										Description: "a list of large or well known software projects that use this package",
										Items: &jsonschema.Definition{
											Type: "string",
										},
									},
									"rank": {
										Type:        jsonschema.Integer,
										Description: "Popularity ranking based on the other results and the number of stars as provided by PyPi.org. Lower number is higher rank.",
									},
								},
							},
						},
					},
				},
			},
		},
		FunctionCall: "auto",
		Stream:       true,
	}

	stream, err := client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		return nil, errs.Wrap(err, "CreateChatCompletionStream failed")
	}
	defer stream.Close()

	output := []string{}
	for {
		response, err := stream.Recv()

		if len(response.Choices) != 0 && response.Choices[0].Delta.FunctionCall != nil {
			// fmt.Printf("%s\n", response.Choices[0].Delta.FunctionCall.Arguments)
			output = append(output, response.Choices[0].Delta.FunctionCall.Arguments)
		}

		if errors.Is(err, io.EOF) {
			// fmt.Println("\nStream finished")
			break
		}

		if err != nil {
			return nil, errs.Wrap(err, "Stream Recv failed")
		}
	}

	// fmt.Println(strings.Join(output, ""))

	var result ChatResult
	if err := json.Unmarshal([]byte(strings.Join(output, "")), &result); err != nil {
		return nil, errs.Wrap(err, "json.Unmarshal failed, value: %s", strings.Join(output, ""))
	}

	return result.Packages, nil
}
