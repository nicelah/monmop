package main

import (
	"fmt"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

var marketNames = map[string]string{
	"^DJI":     "Dow Jones",
	"^GSPC":    "S&P 500",
	"^IXIC":    "NASDAQ",
	"^N225":    "Nikkei",
	"^HSI":     "Hong Kong",
	"^FTSE":    "London",
	"^GDAXI":   "Frankfurt",
	"^TNX":     "10-Year Yield",
	"CAD=X":    "CAD",
	"EURUSD=X": "Euro",
	"RMB=F":    "RMB",
	"CL=F":     "Oil",
	"GC=F":     "Gold",
}

// win
type Win struct {
	w, h, x, y int
}

func (win *Win) print(x, y int, fg, bg termbox.Attribute, s string) (termbox.Attribute, termbox.Attribute) {

	for i := 0; i < len(s); i++ {
		// decodes a utf8 from a string
		// since strings in go are really just immutable []bytes
		r, w := utf8.DecodeRuneInString(s[i:])

		// TOOD: handle escape codes and control characters?

		if x < win.w {
			// fmt.Printf("printing %c\n", r)
			termbox.SetCell(win.x+x, win.y+y, r, fg, bg)
		}

		i += w - 1

		// TOOD: handle tabs

		x += runewidth.RuneWidth(r)
	}

	return fg, bg

}

// thin wrapper around TermBox to provide basic UI for monmop
type Ui struct {
	titleWin             *Win
	marketWin            *Win
	labelWin             *Win
	stockWin             *Win
	commandWin           *Win
	layout               *Layout
	selectedQuote        int
	zerothQuote          int
	selectedVisibleQuote int
	selectedSort         int
	stockQuotes          []Quote
	visibleQuotes        []Quote
	marketQuotes         []Quote
	maxQuotesHeight      int
	profile              *profile
}

func newUI(profile *profile) *Ui {
	wtot, htot := termbox.Size()

	eventQ := make(chan termbox.Event)
	go func() {
		for {
			eventQ <- termbox.PollEvent()
		}
	}()

	eventChann := make(chan termbox.Event)
	go func() {
		for {
			e := <-eventQ
			// TODO
			// handle  alt modifiers?o
			// there's some more work to be done here.
			eventChann <- e
		}
	}()

	return &Ui{
		titleWin: &Win{
			w: wtot,
			h: 1,
			x: 0,
			y: 0,
		},
		marketWin: &Win{
			w: wtot,
			h: 4,
			x: 0,
			y: 1,
		},
		labelWin: &Win{
			w: wtot,
			h: 1,
			x: 0,
			y: 5,
		},
		stockWin: &Win{
			w: wtot,
			h: htot - 7, //terrible practice, but idc
			x: 0,
			y: 6,
		},
		commandWin: &Win{
			w: wtot,
			h: 1,
			x: 0,
			y: htot - 1,
		},
		layout:          NewLayout(),
		selectedQuote:   0,
		selectedSort:    0,
		zerothQuote:     0,
		profile:         profile,
		maxQuotesHeight: htot - 6,
	}

}

func (ui *Ui) draw() {
	fg, bg := termbox.ColorDefault, termbox.ColorDefault

	termbox.Clear(fg, bg)
	ui.drawTitleLine()
	ui.drawMarketWin()
	ui.drawLabelWin()
	ui.drawStockWin()
	ui.drawCommandWin(":add")

	termbox.Flush()
}

// Temp for playing aroudn with termbox
func (ui *Ui) drawTitleLine() {
	fg, bg := termbox.ColorDefault|termbox.AttrBold, termbox.ColorDefault

	currentTime := time.Now()

	titleString := "Monmop by Monta"
	timeString := currentTime.Format(time.UnixDate)

	// %v and -%v for right and left justification respectively
	title := fmt.Sprintf("%-*v%*v", ui.titleWin.w/2, titleString, ui.titleWin.w/2, timeString)

	ui.titleWin.print(0, 0, fg, bg, title)
}

func (ui *Ui) drawLabelWin() {
	fg, bg := termbox.ColorDefault|termbox.AttrUnderline, termbox.ColorDefault

	labels := ""
	for _, col := range ui.layout.columns {
		labels = labels + fmt.Sprintf("%-*v", col.width, col.name)
	}

	ui.labelWin.print(0, 0, fg, bg, labels)
}

func (ui *Ui) drawMarketWin() {
	fg, bg := termbox.ColorDefault, termbox.ColorDefault

	x := 0
	y := 0
	for _, q := range ui.marketQuotes {
		humanFormatted := float2Str(q.LastTrade, 2)
		tickerLine := fmt.Sprintf("%s ", marketNames[q.Ticker], humanFormatted, q.ChangePct)
		if x+len(tickerLine) > ui.marketWin.w {
			y++
			x = 0
		}
		indexLabel := fmt.Sprintf("%s ", marketNames[q.Ticker])
		ui.marketWin.print(x, y, termbox.ColorYellow, bg, indexLabel)
		x += len(indexLabel)
		changeLabel := fmt.Sprintf("%s (%.2f%%)  ", humanFormatted, q.ChangePct)
		ui.marketWin.print(x, y, fg, bg, changeLabel)
		x += len(changeLabel)

	}
}

func (ui *Ui) drawStockWin() {
	_, bg := termbox.ColorDefault, termbox.ColorDefault

	// TODO: don't do this. use a struct with properties, or some constants.
	for id, q := range ui.visibleQuotes {
		tickerLine := ""
		highlightColor := bg

		var lineColor termbox.Attribute
		if q.Change > 0 {
			lineColor = termbox.ColorGreen
		} else if q.Change == 0 {
			lineColor = termbox.ColorBlue
		} else {
			lineColor = termbox.ColorRed
		}

		if ui.selectedVisibleQuote == id {
			highlightColor = termbox.ColorWhite
			lineColor = termbox.ColorBlack
		}

		v := reflect.ValueOf(q)

		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i).Interface()
			val, ok := fieldVal.(float64)
			if ok {
				humanFormatted := float2Str(val, ui.layout.columns[i].precision)
				tickerLine = tickerLine + fmt.Sprintf("%-*v", ui.layout.columns[i].width, humanFormatted)
			} else {
				tickerLine = tickerLine + fmt.Sprintf("%-*v", ui.layout.columns[i].width, v.Field(i).Interface())
			}
		}

		ui.stockWin.print(0, id, lineColor, highlightColor, tickerLine)
	}
}

func (ui *Ui) drawCommandWin(cmd string) {
	fg, bg := termbox.ColorDefault, termbox.ColorDefault

	// TODO:
	ui.commandWin.print(0, 0, fg, bg, fmt.Sprintf("%s ", cmd))
}

func (ui *Ui) GetQuotes() {
	var err error
	ui.stockQuotes, err = FetchQuotes(ui.profile.Tickers)
	if err != nil {
		panic(err)
	}

	ui.marketQuotes, err = FetchMarket()

	if err != nil {
		panic(err)
	}
	// update stock window size to make life easier for us
	// bad practice? idc
	if len(ui.stockQuotes) > ui.maxQuotesHeight {
		ui.stockWin.h = ui.maxQuotesHeight
		ui.visibleQuotes = ui.stockQuotes[:ui.maxQuotesHeight]
	} else {
		ui.stockWin.h = len(ui.stockQuotes)
		ui.visibleQuotes = ui.stockQuotes
	}
}

func (ui *Ui) navigateStockDown() {
	// navigate down a line in the stock window
	updatedPos := ui.selectedQuote + 1
	if updatedPos < ui.stockWin.h {
		ui.selectedQuote = updatedPos
		ui.selectedVisibleQuote += 1
	} else if updatedPos >= ui.stockWin.h && updatedPos < len(ui.stockQuotes) {
		ui.zerothQuote += 1
		ui.visibleQuotes = ui.stockQuotes[updatedPos-ui.stockWin.h+1:]
		ui.selectedQuote = updatedPos
	}
}

func (ui *Ui) navigateStockUp() {
	// navigate up a line in the stock window
	updatedPos := ui.selectedQuote - 1
	if updatedPos >= ui.zerothQuote {
		ui.selectedQuote = updatedPos
		ui.selectedVisibleQuote -= 1
	} else if updatedPos < ui.zerothQuote && ui.zerothQuote > 0 {
		ui.zerothQuote -= 1
		ui.selectedQuote = updatedPos
		ui.visibleQuotes = ui.stockQuotes[ui.zerothQuote : ui.zerothQuote+ui.stockWin.h]
	}
}
