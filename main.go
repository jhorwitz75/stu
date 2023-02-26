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
var hasDefaultHeader bool = false
var status string = ""

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
	if hasDefaultHeader {
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

func setRowSelection(table *tview.Table, row int, selectable bool) {
	for col := 0; col < table.GetColumnCount(); col++ {
		table.GetCell(row, col).SetSelectable(selectable)
	}
}

func resetDefaultHeaderValues(table *tview.Table) {
	for col := 0; col < table.GetColumnCount(); col++ {
		headerCell := tview.NewTableCell(columnToBase26(col)).SetAlign(tview.AlignCenter)
		table.SetCell(0, col, headerCell)
	}
}

func toggleHeaderRow(table *tview.Table) {
	if hasDefaultHeader {
		table.RemoveRow(0)
		hasDefaultHeader = false
	} else {
		table.InsertRow(0)
		resetDefaultHeaderValues(table)
		setRowSelection(table, 1, true)
		hasDefaultHeader = true
	}
	setRowSelection(table, 0, false)
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
			return
		}

		// at first iteration, insert enough columns for new values and
		// update header values if appropriate
		sourceHeaderText := table.GetCell(0, selectedCol).Text
		if row == 1 {
			for i := 0; i < ncols-1; i++ {
				if !hasDefaultHeader {
					newText := fmt.Sprintf("%s.%d", sourceHeaderText, ncols-i)
					table.GetCell(0, selectedCol+i).SetText(newText)
				}
				table.InsertColumn(selectedCol)
				table.GetCell(0, selectedCol).SetSelectable(false)
			}

			// don't forget the last inserted column
			newText := fmt.Sprintf("%s.1", sourceHeaderText)
			table.GetCell(0, selectedCol).SetText(newText)

			// also reset the source column, which has now shifted
			sourceCol = selectedCol + ncols - 1
		}

		// set cell contents to new split values
		for i := 0; i < ncols; i++ {
			col := selectedCol + i
			table.SetCell(row, col, tview.NewTableCell(parts[i]))
		}
	}

	if hasDefaultHeader {
		resetDefaultHeaderValues(table)
	}
}

func pasteContentForm(table *tview.Table) *tview.Form {
	form.AddTextArea("Text", "", 0, 0, 0, nil)
	form.AddButton("Ok", func() {
		text := form.GetFormItem(0).(*tview.TextArea).GetText()
		for i, line := range strings.Split(text, "\n") {
			if line != "" {
				table.SetCell(i, 0, tview.NewTableCell(line))
			}
		}
		toggleHeaderRow(table)
		pages.SwitchToPage("Table")
	})

	return form
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
	if hasDefaultHeader {
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

	// this keeps the table width constant when scrolling
	// TODO: don't do this if row count is too high
	table.SetEvaluateAllRows(true)

	// TODO: add version
	titleBar := tview.NewTextView().
		SetTextColor(tcell.ColorWhite).
		SetText("Stupid Table Utility (STU)")

	statusBar := tview.NewTextView().
		SetTextColor(tcell.ColorGreen).
		SetText(status)

	helpText := tview.NewTextView().
		SetTextColor(tcell.ColorRed).
		SetText("(s) split\n" +
			"(H) toggle header\n" +
			"(L) Move left\n" +
			"(R) Move right\n" +
			"(d) Delete column\n" +
			"(w) Write CSV\n" +
			"(q) Quit")

	confirmationModal := tview.NewModal().AddButtons([]string{"Yes", "No"})

	var flex = tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(titleBar, 1, 0, false).
			AddItem(
				tview.NewFlex().
					AddItem(table, 0, 1, true).
					AddItem(helpText, 0, 1, false), 0, 1, true).
			AddItem(statusBar, 1, 0, false), 0, 1, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if pageName, _ := pages.GetFrontPage(); pageName != "Table" {
			return event
		}

		switch rune := event.Rune(); rune {
		case 'q':
			confirmationModal.
				SetText("Really quit?").
				SetDoneFunc(func(_ int, label string) {
					if label == "Yes" {
						app.Stop()
					}
					pages.HidePage("Confirmation")
				})
			pages.ShowPage("Confirmation")
		case 'd':
			_, col := table.GetSelection()
			colName := table.GetCell(0, col).Text
			confirmationModal.
				SetText(fmt.Sprintf("Really delete column \"%s\"?", colName)).
				SetDoneFunc(func(_ int, label string) {
					if label == "Yes" {
						table.RemoveColumn(col)
					}
					pages.HidePage("Confirmation")
				})
			pages.ShowPage("Confirmation")
		case 'H':
			toggleHeaderRow(table)
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
		statusText := fmt.Sprintf(status)
		statusBar.SetText(statusText)

		return event
	})

	pages.AddPage("PasteContentForm", form, true, false)
	pages.AddPage("SplitColumnByStringForm", form, true, false)
	pages.AddPage("Table", flex, true, true)
	pages.AddPage("Confirmation", confirmationModal, true, false)

	app.SetFocus(flex)

	if len(inputFile) == 0 {
		form.Clear(true)
		pasteContentForm(table)
		pages.SwitchToPage("PasteContentForm")
	} else if strings.HasSuffix(inputFile, "csv") {
		readCSV(table, inputFile)
		toggleHeaderRow(table)
	} else {
		readPlainText(table, inputFile)
		toggleHeaderRow(table)
	}

	if err := app.SetRoot(pages, true).
		EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
