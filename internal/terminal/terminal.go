package terminal

import (
	"fmt"
	"strings"
	"time"
)

const (
	Esc         = "\u001B["
	EraseScreen = Esc + "2J"
	Reset       = Esc + "0m"

	BlackHi   = Esc + "30;1m"
	RedHi     = Esc + "31;1m"
	GreenHi   = Esc + "32;1m"
	YellowHi  = Esc + "33;1m"
	CyanHi    = Esc + "36;1m"
	WhiteHi   = Esc + "37;1m"

	BgGreen  = Esc + "42m"
	BgRed    = Esc + "41m"
	BgBlueHi = Esc + "44;1m"
	BgBlack  = Esc + "40m"
)

// TerminalConfig contains a minimal set of information the renderer needs.
// Keep this small to avoid coupling to the program's dropfile struct.
type TerminalConfig struct {
	BbsName  string
	UserName string
	RealName string
	Terminal string
	Cols     int
	Rows     int
}

// Event represents the minimal event data the renderer requires.
type Event struct {
	Year int
	Text string
}

func MoveCursor(x int, y int) {
	fmt.Printf(Esc+"%d;%df", y, x)
}

func ClearScreen() {
	fmt.Print(EraseScreen)
	MoveCursor(0, 0)
}

func getNumEndingLocal(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return "th"
	}
	switch n % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

// wrapText breaks text into lines that fit within maxWidth (rune-aware).
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	runes := []rune(text)
	if len(runes) <= maxWidth {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var current []rune
	for _, word := range words {
		wr := []rune(word)
		if len(current) == 0 {
			if len(wr) <= maxWidth {
				current = append(current, wr...)
			} else {
				if maxWidth > 3 {
					lines = append(lines, string(wr[:maxWidth-3])+"...")
				} else {
					lines = append(lines, string(wr[:maxWidth]))
				}
			}
			continue
		}
		if len(current)+1+len(wr) <= maxWidth {
			current = append(current, ' ')
			current = append(current, wr...)
		} else {
			lines = append(lines, string(current))
			current = nil
			if len(wr) <= maxWidth {
				current = append(current, wr...)
			} else {
				if maxWidth > 3 {
					lines = append(lines, string(wr[:maxWidth-3])+"...")
				} else {
					lines = append(lines, string(wr[:maxWidth]))
				}
			}
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// RenderEvents draws the header, events, and footer to the terminal.
// It keeps rendering logic isolated so unit tests can target this package.
func RenderEvents(cfg TerminalConfig, events []Event) {
	day := time.Now().Day()
	month := time.Now().Month()
	year := time.Now().Year()
	currentTime := time.Now()

	ClearScreen()

	// Header (kept visually similar to original)
	fmt.Print("\r\n " + BlackHi + Reset + "-" + CyanHi + "---" + GreenHi + "-" + Reset + CyanHi + "--" + GreenHi + "-" + Reset + CyanHi + "-" + GreenHi + "--------- ------------------------------------ ------ -- -  " + Reset)
	fmt.Print("\r\n " + BgGreen + WhiteHi + ">> " + GreenHi + "Glimpse In Time v1.1  " + Reset + BgGreen + BlackHi + ">>" + BgBlack + GreenHi + ">>  " + Reset + WhiteHi + "by " + CyanHi + "<" + WhiteHi + "PHEN0M" + Reset + CyanHi + ">" + Reset)
	fmt.Print("\r\n " + BlackHi + "-" + Reset + CyanHi + "--" + GreenHi + "--" + Reset + CyanHi + "---" + GreenHi + "-" + Reset + CyanHi + "-" + GreenHi + "----- --- -------------------------------- ------ -- -  " + Reset)
	fmt.Printf("\r\n "+BgRed+BlackHi+">>"+BgBlack+" "+"On "+Reset+YellowHi+"THIS DAY"+Reset+", These "+YellowHi+"EVENTS "+Reset+"Happened... "+Reset+RedHi+":: "+Reset+" %v %v%v "+Reset, month, day, getNumEndingLocal(day))
	fmt.Print("\r\n " + BlackHi + "-" + Reset + CyanHi + "--" + GreenHi + "--" + Reset + CyanHi + "---" + GreenHi + "-" + Reset + CyanHi + "-" + GreenHi + "--" + Reset + CyanHi + "--- " + GreenHi + "--- ---------------------------- ------ -- -  " + Reset)

	// Dynamic Event Fitting: available rows and widths are intentionally conservative
	const maxContentRows = 12 // rows 8-19
	const prefixDisplayLength = 10
	const maxLineLength = 75 - prefixDisplayLength

	var selected []Event
	totalRowsUsed := 0
	for _, e := range events {
		wrapped := wrapText(strings.TrimSpace(e.Text), maxLineLength)
		eventRows := len(wrapped) + 1 // +1 blank line
		if totalRowsUsed+eventRows <= maxContentRows && len(selected) < 5 {
			selected = append(selected, e)
			totalRowsUsed += eventRows
		} else {
			break
		}
	}

	// Display selected events starting at row 8
	yPos := 8
	for _, e := range selected {
		yearStr := fmt.Sprintf("%4d", e.Year)
		prefix := " " + CyanHi + yearStr + Reset + CyanHi + " <" + BlackHi + ":" + Reset + CyanHi + "> "
		wrapped := wrapText(strings.TrimSpace(e.Text), maxLineLength)

		MoveCursor(1, yPos)
		fmt.Print(prefix + WhiteHi + wrapped[0] + Reset)
		yPos++
		for i := 1; i < len(wrapped); i++ {
			MoveCursor(1, yPos)
			fmt.Print("          " + WhiteHi + wrapped[i] + Reset)
			yPos++
		}
		// blank line between events
		yPos++
	}

	// Footer
	MoveCursor(1, 20)
	fmt.Print(" " + BlackHi + "-" + Reset + CyanHi + "---" + GreenHi + "-" + Reset + CyanHi + "--" + GreenHi + "-" + Reset + CyanHi + "-" + GreenHi + "-----" + Reset + CyanHi + "-" + GreenHi + "--------------------------------------- ---  --- -- -  " + Reset)
	MoveCursor(1, 21)
	fmt.Printf(" "+BgRed+BlackHi+">>"+BgBlack+" "+WhiteHi+"Generated on %v %v, %v at %v "+Reset, month, day, year, currentTime.Format("3:4 PM"))
	MoveCursor(1, 22)
	fmt.Print(" " + BlackHi + "-" + Reset + CyanHi + "---" + GreenHi + "-" + Reset + CyanHi + "--" + GreenHi + "-" + Reset + CyanHi + "-" + GreenHi + "-----" + Reset + CyanHi + "-" + GreenHi + "--------------------------------------- ---  --- -- -  " + Reset)

	// Pause prompt
	MoveCursor(1, 24)
	fmt.Print("                   " + BgBlueHi + WhiteHi + "<" + Reset + CyanHi + "<  " + BlackHi + "... " + Reset + WhiteHi + "press " + WhiteHi + "ANY KEY " + Reset + WhiteHi + "to " + WhiteHi + "CONTINUE " + Reset + BlackHi + "... " + Reset + CyanHi + ">" + BgBlueHi + WhiteHi + ">" + Reset)
}