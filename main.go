package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"io"
	"net/http"

	"github.com/mattn/go-tty"
)

// Holds a collection of types from Door32.sys dropfile
type Door32Drop struct {
	Node          int
	BbsName       string
	UserName      string
	RealName      string
	SecLevel      int
	TimeLeft      int
	Emulation     int
	CommPort      int
	BaudRate      int
	UserNumber    int
	// Additional terminal capabilities
	Terminal      string
	LoadableFonts bool
	XtendPalette  bool
	Cols          int
	Rows          int
}

const (
	Esc         = "\u001B["
	Osc         = "\u001B]"
	Bel         = "\u0007"
	EraseScreen = Esc + "2J"
	Idle        = 120

	Reset     = Esc + "0m"
	Black     = Esc + "30m"
	Red       = Esc + "31m"
	Green     = Esc + "32m"
	Yellow    = Esc + "33m"
	Blue      = Esc + "34m"
	Magenta   = Esc + "35m"
	Cyan      = Esc + "36m"
	White     = Esc + "37m"
	BlackHi   = Esc + "30;1m"
	RedHi     = Esc + "31;1m"
	GreenHi   = Esc + "32;1m"
	YellowHi  = Esc + "33;1m"
	BlueHi    = Esc + "34;1m"
	MagentaHi = Esc + "35;1m"
	CyanHi    = Esc + "36;1m"
	WhiteHi   = Esc + "37;1m"

	BgBlack     = Esc + "40m"
	BgRed       = Esc + "41m"
	BgGreen     = Esc + "42m"
	BgYellow    = Esc + "43m"
	BgBlue      = Esc + "44m"
	BgMagenta   = Esc + "45m"
	BgCyan      = Esc + "46m"
	BgWhite     = Esc + "47m"
	BgBlackHi   = Esc + "40;1m"
	BgRedHi     = Esc + "41;1m"
	BgGreenHi   = Esc + "42;1m"
	BgYellowHi  = Esc + "43;1m"
	BgBlueHi    = Esc + "44;1m"
	BgMagentaHi = Esc + "45;1m"
	BgCyanHi    = Esc + "46;1m"
	BgWhiteHi   = Esc + "47;1m"
)

var (
	Pd       Door32Drop
	DropPath string
)

// NewTimer boots a user after being idle too long
func NewTimer(seconds int, action func()) *time.Timer {
	timer := time.NewTimer(time.Second * time.Duration(seconds))

	go func() {
		<-timer.C
		action()
	}()
	return timer
}

// DetectTerminalCapabilities detects terminal type and capabilities based on environment
func DetectTerminalCapabilities() (string, bool, bool, int, int) {
	var terminal string
	var loadableFonts bool
	var xtendPalette bool
	var cols, rows int = 80, 25 // default values
	
	// Get terminal type from environment variables
	termType := strings.ToLower(os.Getenv("TERM"))
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	
	// Try to get terminal size from environment
	if colsStr := os.Getenv("COLUMNS"); colsStr != "" {
		if c, err := strconv.Atoi(colsStr); err == nil {
			cols = c
		}
	}
	if rowsStr := os.Getenv("LINES"); rowsStr != "" {
		if r, err := strconv.Atoi(rowsStr); err == nil {
			rows = r
		}
	}
	
	// Detect terminal capabilities based on TERM environment or program
	if termType == "ansi-256color-rgb" || cols > 80 {
		terminal = "Netrunner"
	} else if termProgram == "syncterm" || termType == "syncterm" {
		terminal = "Syncterm"
	} else if termProgram == "magiterm" || termType == "magiterm" {
		terminal = "Magiterm"
	} else {
		terminal = "ANSI-Term"
	}
	
	// Set capabilities based on terminal type
	if terminal == "Netrunner" || terminal == "ANSI-Term" || terminal == "Magiterm" {
		loadableFonts = false
	} else {
		loadableFonts = true
	}
	
	if terminal == "Syncterm" || terminal == "Netrunner" || terminal == "Magiterm" {
		xtendPalette = true
	} else {
		xtendPalette = false
	}
	
	return terminal, loadableFonts, xtendPalette, cols, rows
}

// Move cursor to X, Y location
func MoveCursor(x int, y int) {
	fmt.Printf(Esc+"%d;%df", y, x)
}

// Erase the screen
func ClearScreen() {
	fmt.Print(EraseScreen)
	MoveCursor(0, 0)
}

// Returns door32.sys values as strings: commport, baudind, baudrate, bbsname, usernum, realname, username, seclevel, timeleft, emulation, node
func DropFileData(path string) (string, string, string, string, string, string, string, string, string, string, string) {
	var commport string
	var baudind string
	var baudrate string
	var bbsname string
	var usernum string
	var realname string
	var username string
	var seclevel string
	var timeleft string
	var emulation string
	var node string

	file, err := os.Open(strings.ToLower(path + "/door32.sys"))
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var text []string

	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	count := 0
	for _, line := range text {
		if count == 0 {
			commport = line
		}
		if count == 1 {
			baudind = line
		}
		if count == 2 {
			baudrate = line
		}
		if count == 3 {
			bbsname = line
		}
		if count == 4 {
			usernum = line
		}
		if count == 5 {
			realname = line
		}
		if count == 6 {
			username = line
		}
		if count == 7 {
			seclevel = line
		}
		if count == 8 {
			timeleft = line
		}
		if count == 9 {
			emulation = line
		}
		if count == 10 {
			node = line
		}
		if count == 11 {
			break
		}
		count++
		continue
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return commport, baudind, baudrate, bbsname, usernum, realname, username, seclevel, timeleft, emulation, node
}

// Print text at an X, Y location
func PrintStringLoc(text string, x int, y int) {
	yLoc := y
	s := bufio.NewScanner(strings.NewReader(text))
	for s.Scan() {
		fmt.Fprintf(os.Stdout, Esc+strconv.Itoa(yLoc)+";"+strconv.Itoa(x)+"f"+s.Text())
		yLoc++
	}
}

func getNumEnding() string {

	dayStr := time.Now().Day()

	if dayStr-1 == 1 && len(fmt.Sprint(dayStr)) == 1 {
		return "st"
	} else if dayStr-1 == 2 && dayStr-2 != 12 {
		return "nd"
	} else if dayStr-1 == 3 {
		return "rd"
	} else {
		return "th"
	}
}

// WikimediaEvent represents an event from the Wikimedia API
type WikimediaEvent struct {
	Year int    `json:"year"`
	Text string `json:"text"`
	Type string `json:"type"`
}

// WikimediaResponse represents the full response from Wikimedia API
type WikimediaResponse struct {
	Events []WikimediaEvent `json:"events"`
	Births []WikimediaEvent `json:"births"`
	Deaths []WikimediaEvent `json:"deaths"`
}

// wrapText breaks text into lines that fit within maxWidth
func wrapText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}
	
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	
	var lines []string
	var currentLine strings.Builder
	
	for _, word := range words {
		testLine := currentLine.String()
		if testLine == "" {
			testLine = word
		} else {
			testLine += " " + word
		}
		
		if len(testLine) <= maxWidth {
			if currentLine.Len() > 0 {
				currentLine.WriteString(" ")
			}
			currentLine.WriteString(word)
		} else {
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			} else {
				// Word is too long, break it
				if len(word) > maxWidth-3 {
					lines = append(lines, word[:maxWidth-3]+"...")
				} else {
					lines = append(lines, word)
				}
			}
		}
	}
	
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}
	
	if len(lines) == 0 {
		return []string{""}
	}
	
	return lines
}

func displayLoadingAnimation(done <-chan bool) {
	loadingSteps := []struct {
		bar   string
		delay int
	}{
		{
			bar:   " " + Cyan + "\xDB\xDB\xDB\xDB" + Reset + "\xB0\xB0\xB0\xB0\xB0\xB0 " + Green + "Fetching historical data" + Reset,
			delay: 300,
		},
		{
			bar:   " " + Cyan + "\xDB\xDB\xDB\xDB\xDB\xDB" + Reset + "\xB0\xB0\xB0\xB0 " + Green + "Processing events" + Reset,
			delay: 400,
		},
		{
			bar:   " " + Cyan + "\xDB\xDB\xDB\xDB\xDB\xDB\xDB\xDB" + Reset + "\xB0\xB0 " + Green + "Applying filters and sorting" + Reset,
			delay: 600,
		},
		{
			bar:   " " + Cyan + "\xDB\xDB\xDB\xDB\xDB\xDB\xDB\xDB\xDB\xDB " + Green + "Ready to display" + Reset,
			delay: 300,
		},
	}
	
	loadingBarRow := 12
	stepIndex := 0
	
	// Keep cycling through animation until done
	for {
		select {
		case <-done:
			// Clear the loading bar when done
			MoveCursor(1, loadingBarRow)
			fmt.Print(Esc + "K") // Clear the loading bar
			return
		case <-time.After(time.Duration(loadingSteps[stepIndex].delay) * time.Millisecond):
			MoveCursor(1, loadingBarRow)
			fmt.Print(Esc + "K") // Clear the line
			fmt.Print(loadingSteps[stepIndex].bar)
			stepIndex = (stepIndex + 1) % len(loadingSteps) // Cycle through steps
		}
	}
}

func fetchHistoricalEvents() ([]WikimediaEvent, error) {
	now := time.Now()
	month := fmt.Sprintf("%02d", int(now.Month()))
	day := fmt.Sprintf("%02d", now.Day())
	
	url := fmt.Sprintf("https://api.wikimedia.org/feed/v1/wikipedia/en/onthisday/all/%s/%s", month, day)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", "Go Day-in-History BBS Door/1.0 (github.com/robbiew/history)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %v", err)
	}
	defer resp.Body.Close()
	
	// Handle redirects
	if resp.StatusCode == 301 || resp.StatusCode == 302 {
		return nil, fmt.Errorf("API redirect not implemented")
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	var wikimediaResp WikimediaResponse
	err = json.Unmarshal(body, &wikimediaResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	
	// Combine all events and mark their types
	var allEvents []WikimediaEvent
	
	// Process events
	for _, event := range wikimediaResp.Events {
		event.Type = "event"
		allEvents = append(allEvents, event)
	}
	
	// Process births (optional, we'll exclude them for cleaner results)
	// for _, birth := range wikimediaResp.Births {
	// 	birth.Type = "birth"
	// 	allEvents = append(allEvents, birth)
	// }
	
	// Process deaths (optional, we'll exclude them for cleaner results)
	// for _, death := range wikimediaResp.Deaths {
	// 	death.Type = "death"
	// 	allEvents = append(allEvents, death)
	// }
	
	// Random selection strategy - shuffle the events
	if len(allEvents) > 1 {
		for i := len(allEvents) - 1; i > 0; i-- {
			j := time.Now().UnixNano() % int64(i+1)
			allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
		}
	}
	
	return allEvents, nil
}

func generateEventList() {
	day := time.Now().Day()
	month := time.Now().Month()
	year := time.Now().Year()
	current_time := time.Now()

	ClearScreen()

	// Display header (rows 1-7)
	fmt.Print("\r\n " + BlackHi + Reset + "-" + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "--------- ------------------------------------ ------ -- -  " + Reset)
	fmt.Print("\r\n " + BgGreen + WhiteHi + ">> " + GreenHi + "Glimpse In Time v1.1  " + Reset + BgGreen + Black + ">>" + BgBlack + Green + ">>  " + Reset + WhiteHi + "by " + CyanHi + "Smooth " + Reset + Cyan + "<" + WhiteHi + "PHEN0M" + Reset + Cyan + ">" + Reset)
	fmt.Print("\r\n " + BlackHi + "-" + Reset + Cyan + "--" + GreenHi + "--" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "----- --- -------------------------------- ------ -- -  " + Reset)
	fmt.Printf("\r\n "+BgRed+Black+">>"+BgBlack+" "+MagentaHi+"On "+Reset+YellowHi+"THIS DAY"+MagentaHi+", These "+YellowHi+"EVENTS "+MagentaHi+"Happened... "+Reset+Red+":: "+Yellow+" %v %v%v "+Red+" ::"+Reset, month, day, getNumEnding())
	fmt.Print("\r\n " + BlackHi + "-" + Reset + Cyan + "--" + GreenHi + "--" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "--" + Reset + Cyan + "--- " + GreenHi + "--- ---------------------------- ------ -- -  " + Reset)

	// Start loading animation in background and fetch events concurrently
	done := make(chan bool)
	go displayLoadingAnimation(done)
	
	// Fetch events from Wikimedia API
	events, err := fetchHistoricalEvents()
	
	// Stop the loading animation
	done <- true
	close(done)
	if err != nil {
		// Clear screen and redraw header for error display
		ClearScreen()
		fmt.Print("\r\n " + BlackHi + Reset + "-" + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "--------- ------------------------------------ ------ -- -  " + Reset)
		fmt.Print("\r\n " + BgGreen + WhiteHi + ">> " + GreenHi + "Glimpse In Time v1.1  " + Reset + BgGreen + Black + ">>" + BgBlack + Green + ">>  " + Reset + WhiteHi + "by " + CyanHi + "Smooth " + Reset + Cyan + "<" + WhiteHi + "PHEN0M" + Reset + Cyan + ">" + Reset)
		fmt.Print("\r\n " + BlackHi + "-" + Reset + Cyan + "--" + GreenHi + "--" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "----- --- -------------------------------- ------ -- -  " + Reset)
		fmt.Printf("\r\n "+BgRed+Black+">>"+BgBlack+" "+MagentaHi+"On "+Reset+YellowHi+"THIS DAY"+MagentaHi+", These "+YellowHi+"EVENTS "+MagentaHi+"Happened... "+Reset+Red+":: "+Yellow+" %v %v%v "+Red+" ::"+Reset, month, day, getNumEnding())
		fmt.Print("\r\n " + BlackHi + "-" + Reset + Cyan + "--" + GreenHi + "--" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "--" + Reset + Cyan + "--- " + GreenHi + "--- ---------------------------- ------ -- -  " + Reset)
		
		MoveCursor(1, 8)
		fmt.Printf(RedHi+"Error fetching events: %v"+Reset+"\r\n", err)
		fmt.Print(WhiteHi+"Please check your internet connection and try again."+Reset+"\r\n")
		
		// Display footer at rows 20-22
		MoveCursor(1, 20)
		fmt.Print(" " + BlackHi + "-" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "-----" + Reset + Cyan + "-" + GreenHi + "--------------------------------------- ---  --- -- -  " + Reset)
		MoveCursor(1, 21)
		fmt.Printf(" "+BgRed+Black+">>"+BgBlack+" "+WhiteHi+"Generated on %v %v, %v at %v "+Cyan+"(error)"+Reset, month, day, year, current_time.Format("3:4 PM"))
		MoveCursor(1, 22)
		fmt.Print(" " + BlackHi + "-" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "-----" + Reset + Cyan + "-" + GreenHi + "--------------------------------------- ---  --- -- -  " + Reset)
		
		MoveCursor(1, 24)
		fmt.Print("                   " + BgBlueHi + WhiteHi + "<" + Reset + Cyan + "<  " + BlackHi + "... " + Reset + White + "press " + WhiteHi + "ANY KEY " + Reset + White + "to " + WhiteHi + "CONTINUE " + Reset + BlackHi + "... " + Reset + Cyan + ">" + BgBlue + WhiteHi + ">" + Reset)
		return
	}

	if len(events) == 0 {
		// Clear screen and redraw header for no events display
		ClearScreen()
		fmt.Print("\r\n " + BlackHi + Reset + "-" + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "--------- ------------------------------------ ------ -- -  " + Reset)
		fmt.Print("\r\n " + BgGreen + WhiteHi + ">> " + GreenHi + "Glimpse In Time v1.1  " + Reset + BgGreen + Black + ">>" + BgBlack + Green + ">>  " + Reset + WhiteHi + "by " + CyanHi + "Smooth " + Reset + Cyan + "<" + WhiteHi + "PHEN0M" + Reset + Cyan + ">" + Reset)
		fmt.Print("\r\n " + BlackHi + "-" + Reset + Cyan + "--" + GreenHi + "--" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "----- --- -------------------------------- ------ -- -  " + Reset)
		fmt.Printf("\r\n "+BgRed+Black+">>"+BgBlack+" "+MagentaHi+"On "+Reset+YellowHi+"THIS DAY"+MagentaHi+", These "+YellowHi+"EVENTS "+MagentaHi+"Happened... "+Reset+Red+":: "+Yellow+" %v %v%v "+Red+" ::"+Reset, month, day, getNumEnding())
		fmt.Print("\r\n " + BlackHi + "-" + Reset + Cyan + "--" + GreenHi + "--" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "--" + Reset + Cyan + "--- " + GreenHi + "--- ---------------------------- ------ -- -  " + Reset)
		
		MoveCursor(1, 8)
		fmt.Print(YellowHi + "No historical events found for today." + Reset + "\r\n")
		
		// Display footer at rows 20-22
		MoveCursor(1, 20)
		fmt.Print(" " + BlackHi + "-" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "-----" + Reset + Cyan + "-" + GreenHi + "--------------------------------------- ---  --- -- -  " + Reset)
		MoveCursor(1, 21)
		fmt.Printf(" "+BgRed+Black+">>"+BgBlack+" "+WhiteHi+"Generated on %v %v, %v at %v "+Cyan+"(no events)"+Reset, month, day, year, current_time.Format("3:4 PM"))
		MoveCursor(1, 22)
		fmt.Print(" " + BlackHi + "-" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "-----" + Reset + Cyan + "-" + GreenHi + "--------------------------------------- ---  --- -- -  " + Reset)
		
		MoveCursor(1, 24)
		fmt.Print("                   " + BgBlueHi + WhiteHi + "<" + Reset + Cyan + "<  " + BlackHi + "... " + Reset + White + "press " + WhiteHi + "ANY KEY " + Reset + White + "to " + WhiteHi + "CONTINUE " + Reset + BlackHi + "... " + Reset + Cyan + ">" + BgBlue + WhiteHi + ">" + Reset)
		return
	}

	// Clear screen completely and redraw header before displaying events
	ClearScreen()
	
	// Redraw header (rows 1-7)
	fmt.Print("\r\n " + BlackHi + Reset + "-" + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "--------- ------------------------------------ ------ -- -  " + Reset)
	fmt.Print("\r\n " + BgGreen + WhiteHi + ">> " + GreenHi + "Glimpse In Time v1.1  " + Reset + BgGreen + Black + ">>" + BgBlack + Green + ">>  " + Reset + WhiteHi + "by " + CyanHi + "Smooth " + Reset + Cyan + "<" + WhiteHi + "PHEN0M" + Reset + Cyan + ">" + Reset)
	fmt.Print("\r\n " + BlackHi + "-" + Reset + Cyan + "--" + GreenHi + "--" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "----- --- -------------------------------- ------ -- -  " + Reset)
	fmt.Printf("\r\n "+BgRed+Black+">>"+BgBlack+" "+MagentaHi+"On "+Reset+YellowHi+"THIS DAY"+MagentaHi+", These "+YellowHi+"EVENTS "+MagentaHi+"Happened... "+Reset+Red+":: "+Yellow+" %v %v%v "+Red+" ::"+Reset, month, day, getNumEnding())
	fmt.Print("\r\n " + BlackHi + "-" + Reset + Cyan + "--" + GreenHi + "--" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "--" + Reset + Cyan + "--- " + GreenHi + "--- ---------------------------- ------ -- -  " + Reset)

	// Dynamic Event Fitting: Calculate available screen space (rows 8-19 = 12 rows)
	const maxContentRows = 12 // Rows 8-19
	const prefixDisplayLength = 10 // " YYYY <:> " = 10 characters
	const maxLineLength = 75 - prefixDisplayLength // Leave room for prefix
	
	var selectedEvents []WikimediaEvent
	totalRowsUsed := 0
	
	// Fit as many events as possible in available space
	for _, event := range events {
		// Word wrap the event text
		wrappedLines := wrapText(strings.TrimSpace(event.Text), maxLineLength)
		eventRows := len(wrappedLines) + 1 // +1 for blank line after event
		
		// Check if this event will fit
		if totalRowsUsed+eventRows <= maxContentRows && len(selectedEvents) < 5 {
			selectedEvents = append(selectedEvents, event)
			totalRowsUsed += eventRows
		} else {
			break // No more room
		}
	}

	// Display events starting at row 8
	yPos := 8
	for _, event := range selectedEvents {
		// Format year with consistent 4-digit padding
		yearStr := fmt.Sprintf("%4d", event.Year)
		prefix := " " + CyanHi + yearStr + Reset + Cyan + " <" + BlackHi + ":" + Reset + Cyan + "> "
		
		// Word wrap the event text
		wrappedLines := wrapText(strings.TrimSpace(event.Text), maxLineLength)
		
		// Display first line with prefix
		MoveCursor(1, yPos)
		fmt.Print(prefix + WhiteHi + wrappedLines[0] + Reset)
		yPos++
		
		// Display continuation lines with proper indentation (10 spaces to align with text)
		for i := 1; i < len(wrappedLines); i++ {
			MoveCursor(1, yPos)
			fmt.Print("          " + WhiteHi + wrappedLines[i] + Reset)
			yPos++
		}
		
		// Add blank line between events
		yPos++
	}

	// Display footer at rows 20-22
	MoveCursor(1, 20)
	fmt.Print(" " + BlackHi + "-" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "-----" + Reset + Cyan + "-" + GreenHi + "--------------------------------------- ---  --- -- -  " + Reset)
	MoveCursor(1, 21)
	fmt.Printf(" "+BgRed+Black+">>"+BgBlack+" "+WhiteHi+"Generated on %v %v, %v at %v "+Cyan+"(random)"+Reset, month, day, year, current_time.Format("3:4 PM"))
	MoveCursor(1, 22)
	fmt.Print(" " + BlackHi + "-" + Reset + Cyan + "---" + GreenHi + "-" + Reset + Cyan + "--" + GreenHi + "-" + Reset + Cyan + "-" + GreenHi + "-----" + Reset + Cyan + "-" + GreenHi + "--------------------------------------- ---  --- -- -  " + Reset)

	// Pause message at row 24
	MoveCursor(1, 24)
	fmt.Print("                   " + BgBlueHi + WhiteHi + "<" + Reset + Cyan + "<  " + BlackHi + "... " + Reset + White + "press " + WhiteHi + "ANY KEY " + Reset + White + "to " + WhiteHi + "CONTINUE " + Reset + BlackHi + "... " + Reset + Cyan + ">" + BgBlue + WhiteHi + ">" + Reset)
}

func init() {
	// Use FLAG to get command line paramenters
	pathPtr := flag.String("path", "", "path to node directory")
	required := []string{"path"}

	flag.Parse()

	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			// or possibly use `log.Fatalf` instead of:
			fmt.Fprintf(os.Stderr, "missing path to node directory, e.g.: ./phenomdroptest -%s /bbs/temp/1 \n", req)
			os.Exit(2) // the same exit code flag.Parse uses
		}
	}

	// read the drop file and save to struct
	DropPath = *pathPtr
	commport, _, baudrate, bbsname, usernum, realname, username, seclevel, timeleft, emulation, node := DropFileData(DropPath)

	// convert some values to int
	intnode, _ := strconv.Atoi(node)
	intcommport, _ := strconv.Atoi(commport)
	intbaudrate, _ := strconv.Atoi(baudrate)
	intusernum, _ := strconv.Atoi(usernum)
	intseclevel, _ := strconv.Atoi(seclevel)
	inttimeleft, _ := strconv.Atoi(timeleft)
	intemulation, _ := strconv.Atoi(emulation)

	// detect terminal capabilities
	terminal, loadableFonts, xtendPalette, cols, rows := DetectTerminalCapabilities()

	// assign to struct
	Pd = Door32Drop{
		Node:          intnode,
		BbsName:       bbsname,
		UserName:      username,
		RealName:      realname,
		SecLevel:      intseclevel,
		TimeLeft:      inttimeleft,
		Emulation:     intemulation,
		CommPort:      intcommport,
		BaudRate:      intbaudrate,
		UserNumber:    intusernum,
		Terminal:      terminal,
		LoadableFonts: loadableFonts,
		XtendPalette:  xtendPalette,
		Cols:          cols,
		Rows:          rows,
	}
}

func main() {
	// Start the idle timer
	shortTimer := NewTimer(Idle, func() {
		fmt.Println("\r\nYou've been idle for too long... exiting!")
		time.Sleep(1 * time.Second)
		os.Exit(0)
	})
	defer shortTimer.Stop()

	ClearScreen()
	MoveCursor(0, 0)

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	for {

		generateEventList()
		_, err := tty.ReadRune()
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)

	}
}
