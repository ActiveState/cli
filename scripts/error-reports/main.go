package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/davidji99/rollrest-go/rollrest"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"golang.org/x/crypto/ssh/terminal"
)

var termWidth int
var termHeight int

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var err error
	termWidth, termHeight, err = terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}

	token := os.Getenv("ROLLBAR_ACCESS_TOKEN")
	if token == "" {
		return fmt.Errorf("ROLLBAR_ACCESS_TOKEN env var should be set")
	}

	client, err := rollrest.New(rollrest.AuthAAT(token), rollrest.UserAgent("rollbar-go-custom"))
	if err != nil {
		return err
	}

	if err := ui.Init(); err != nil {
		panic("Could not init UI")
	}
	defer ui.Close()

	progressMsg("Loading data from rollbar")

	res, err := client.RQL.Run(`
		SELECT COUNT(*) as C, item.counter
		FROM item_occurrence
		WHERE 
		    item.level=50
		GROUP BY body.trace_chain.0.exception.message, person.id
		ORDER BY timestamp DESC
		LIMIT 100
	`)
	if err != nil {
		return err
	}

	results := groupResults(unmarshalResult(res))

	return renderGroupedEntries(results)
}

type Entry struct {
	ID        int
	Title     string
	Count     int
	PersonID  string
	Timestamp time.Time
}

type GroupedEntry struct {
	Entry
	Persons  map[string]struct{}
	Children []Entry
}

func unmarshalResult(res *rollrest.RqlJobResult) []Entry {
	progressMsg("Unmarshalling results")

	var result []Entry
	for _, row := range res.Rows {
		result = append(result, Entry{
			int(row[1].(float64)),
			row[2].(string),
			int(row[0].(float64)),
			row[3].(string),
			time.Unix(int64(row[4].(float64)), 0),
		})
	}
	return result
}

func groupResults(entries []Entry) []GroupedEntry {
	var result []GroupedEntry
	for x, entry := range entries {
		progressMsg(fmt.Sprintf("Grouping: %d/%d", x+1, len(entries)))

		matched := false
		for x, entry2 := range result {
			sim := strutil.Similarity(entry.Title, entry2.Title, metrics.NewLevenshtein())
			if sim > 0.7 {
				result[x].Count = entry2.Count + entry.Count
				result[x].Persons[entry.PersonID] = struct{}{}
				result[x].Children = append(result[x].Children, entry)
				matched = true
			}
		}
		if !matched {
			result = append(result, GroupedEntry{entry, map[string]struct{}{entry.PersonID: struct{}{}}, []Entry{}})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

func progressMsg(msg string) {
	p := widgets.NewParagraph()
	p.Border = false
	p.Title = msg
	p.SetRect(0, 0, termWidth, termHeight)
	ui.Render(p)
}
