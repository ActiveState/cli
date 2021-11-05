package main

import (
	"fmt"
	"strconv"

	"github.com/ActiveState/cli/scripts/error-reports/cwidgets"
	ui "github.com/gizak/termui/v3"
	"github.com/skratchdot/open-golang/open"
)

var itemURL = `https://rollbar.com/activestate/state-tool/items/%d`

func renderGroupedEntries(entries []GroupedEntry) error {
	rows := [][]string{}
	for i, row := range entries {
		rows = append(rows, []string{strconv.Itoa(i), row.Title, strconv.Itoa(row.Count), strconv.Itoa(len(row.Persons))})
	}
	return renderTable([]Column{
		{"", 5},
		{"Title", termWidth - 25},
		{"Count", 10},
		{"People", 10},
	}, rows, func(idx int) { renderGroupedEntry(entries[idx]) })
}

func renderGroupedEntry(entry GroupedEntry) error {
	rows := [][]string{}
	for i, row := range entry.Children {
		rows = append(rows, []string{strconv.Itoa(i), row.Title, strconv.Itoa(row.Count)})
	}
	return renderTable([]Column{
		{"", 5},
		{"Title", termWidth - 25},
		{"Count (by person)", 20},
	}, rows, func(idx int) {
		open.Run(fmt.Sprintf(itemURL, entry.Children[idx].ID))
	})
}

type Column struct {
	value string
	width int
}

func renderTable(columns []Column, rows [][]string, onSelect func(int)) error {
	tbl := cwidgets.NewTable()
	colvalues := []string{}
	for _, column := range columns {
		colvalues = append(colvalues, column.value)
		tbl.ColumnWidths = append(tbl.ColumnWidths, column.width)
	}
	tbl.Rows = append(tbl.Rows, colvalues)
	tbl.Border = false
	tbl.Rows = append(tbl.Rows, rows...)
	tbl.SetRect(0, 0, termWidth, termHeight)

	if len(tbl.Rows) == 0 {
		return fmt.Errorf("No results")
	}

	ui.Render(tbl)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return nil
		case "j", "<Down>":
			tbl.ScrollDown()
		case "k", "<Up>":
			tbl.ScrollUp()
		case "<C-d>":
			tbl.ScrollHalfPageDown()
		case "<C-u>":
			tbl.ScrollHalfPageUp()
		case "<C-f>":
			tbl.ScrollPageDown()
		case "<C-b>":
			tbl.ScrollPageUp()
		case "<Home>":
			tbl.ScrollTop()
		case "G", "<End>":
			tbl.ScrollBottom()
		case "<Enter>":
			if onSelect != nil {
				onSelect(tbl.SelectedRow - 1)
			}
		}

		ui.Render(tbl)
	}
}
