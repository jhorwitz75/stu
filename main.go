package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/gen2brain/beeep"
	"github.com/rivo/tview"
)

var app = tview.NewApplication()
var pages = tview.NewPages()
var form = tview.NewForm()
var splitString string = ""
var splitCols int = 0
var hasHeader bool = false

// adapted from github.com/jason-meredith/warships
func columnToBase26(col int) string {
	results := make([]rune, 0)

	var remainder int

	if col == 0 {
		return "A"
	}

	for col > 0 {
		remainder = col % 26
		col = col / 26
		results = append(results, rune(remainder+'A'))
	}

	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return string(results)
}

func swapColumns(table *tview.Table, leftCol int, rightCol int) {
	var startRow int
	if hasHeader {
		startRow = 1
	} else {
		startRow = 0
	}

	for row := startRow; row < table.GetRowCount(); row++ {
		rightCell := table.GetCell(row, rightCol)
		temp := table.GetCell(row, leftCol)
		table.SetCell(row, rightCol, temp)
		table.SetCell(row, leftCol, rightCell)
	}
}

func disableHeaderSelection(table *tview.Table) {
	for col := 0; col < table.GetColumnCount(); col++ {
		table.GetCell(0, col).SetSelectable(false)
	}
}

func toggleHeaderRow(table *tview.Table) {
	if hasHeader {
		table.RemoveRow(0)
		hasHeader = false
	} else {
		table.InsertRow(0)
		for col := 0; col < table.GetColumnCount(); col++ {
			headerCell := tview.NewTableCell(columnToBase26(col)).SetAlign(tview.AlignCenter)
			table.SetCell(0, col, headerCell)
		}
		hasHeader = true
	}
}

func splitColumnByString(table *tview.Table, s string) {
	_, selectedCol := table.GetSelection()
	sourceCol := selectedCol
	for row := 1; row < table.GetRowCount(); row++ {
		// get cell contents to split
		cell := table.GetCell(row, sourceCol)

		// split cell contents and determine # of new columns
		var parts []string
		if splitCols == 0 {
			parts = strings.Split(cell.Text, s)
		} else {
			parts = strings.SplitN(cell.Text, s, splitCols)
			if len(parts) < splitCols {
				beep()
				break
			}
		}
		ncols := len(parts)

		// stop if we have nothing to split
		if ncols == 1 {
			beep()
			break
		}

		// at first iteration, insert enough columns for new values
		if row == 1 {
			for i := 0; i < ncols-1; i++ {
				table.InsertColumn(selectedCol)
				table.GetCell(0, i).SetSelectable(false)
			}
			// also reset the source column, which has now shifted
			sourceCol = selectedCol + ncols - 1
		}

		// set cell contents to new split values
		for i := 0; i < ncols; i++ {
			col := selectedCol + i
			table.SetCell(row, col, tview.NewTableCell(parts[i]))
		}
	}
}

func splitColumnByStringForm(table *tview.Table) *tview.Form {
	form.AddInputField("Split on string", splitString, 60, nil, func(value string) {
		splitString = value
	})
	form.AddInputField("Maximum fields (0 to disable)", "0", 2, nil, func(value string) {
		i, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}
		splitCols = i
	})
	form.AddButton("Split", func() {
		splitColumnByString(table, splitString)
		pages.SwitchToPage("Table")
	})
	form.AddButton("Cancel", func() {
		pages.SwitchToPage("Table")
	})

	return form
}

func beep() {
	beeep.Beep(440.0, 200)
}

func readCSV(table *tview.Table, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	csvReader := csv.NewReader(file)
	var rowNum int
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		for colNum, field := range row {
			table.SetCell(rowNum, colNum, tview.NewTableCell(field))
		}
		rowNum++
	}
}

func writeCSV(table *tview.Table, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	ncols := table.GetColumnCount()

	var startRow int
	if hasHeader {
		startRow = 1
	} else {
		startRow = 0
	}

	for row := startRow; row < table.GetRowCount(); row++ {
		record := make([]string, ncols)
		for col := 0; col < ncols; col++ {
			record[col] = table.GetCell(row, col).Text
		}
		if err := w.Write(record); err != nil {
			panic(err)
		}
	}
}

func readPlainText(table *tview.Table, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var numLines int
	for scanner.Scan() {
		table.SetCell(numLines, 0, tview.NewTableCell(scanner.Text()))
		numLines++
	}
}

func main() {
	flag.Parse()

	inputFile := flag.Arg(0)

	table := tview.NewTable().SetBorders(true)
	table.SetSelectable(true, true)
	table.SetFixed(1, 0)

	if inputFile == "" {
		table.SetCell(0, 0, tview.NewTableCell(""))
	} else if strings.HasSuffix(inputFile, "csv") {
		readCSV(table, inputFile)
	} else {
		readPlainText(table, inputFile)
	}

	toggleHeaderRow(table)
	disableHeaderSelection(table)

	// TODO: add version
	titleBar := tview.NewTextView().
		SetTextColor(tcell.ColorWhite).
		SetText("Stupid Table Utility (STU)")

	statusBar := tview.NewTextView().
		SetTextColor(tcell.ColorGreen).
		SetText("Hello world!")

	helpText := tview.NewTextView().
		SetTextColor(tcell.ColorRed).
		SetText("(s) split\n(H) toggle header row\n(L) Move left\n(R) Move right\n(w) Write CSV\n(q) Quit")

	var flex = tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(titleBar, 1, 0, false).
			AddItem(
				tview.NewFlex().
					AddItem(table, 0, 1, true).
					AddItem(helpText, 0, 1, false), 0, 1, true).
			AddItem(statusBar, 1, 0, false), 0, 1, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch rune := event.Rune(); rune {
		case 'q':
			app.Stop()
		case 'H':
			toggleHeaderRow(table)
			disableHeaderSelection(table)
		case 'L':
			row, col := table.GetSelection()
			if col > 0 {
				swapColumns(table, col-1, col)
				table.Select(row, col-1)
			} else {
				beep()
			}
			return nil
		case 'R':
			row, col := table.GetSelection()
			if col < table.GetColumnCount()-1 {
				swapColumns(table, col, col+1)
				table.Select(row, col+1)
			} else {
				beep()
			}
			return nil
		case 's':
			form.Clear(true)
			splitColumnByStringForm(table)
			pages.SwitchToPage("SplitColumnByStringForm")
			return nil
		case 'w':
			writeCSV(table, inputFile)
			app.Stop()
		}

		// update status bar
		statusText := fmt.Sprintf("Hi there. Key code is %d", event.Rune())
		statusBar.SetText(statusText)

		return event
	})

	pages.AddPage("Table", flex, true, true)
	pages.AddPage("SplitColumnByStringForm", form, true, false)

	app.SetFocus(flex)

	if err := app.SetRoot(pages, true).
		EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
