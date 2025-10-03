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
	"context"
	"sync"
	"math/rand"
	"path/filepath"
	"sort"
 
	"encoding/json"
	"io"
	"net/http"
 
	"github.com/mattn/go-tty"
	"github.com/robbiew/history/internal/terminal"
	"github.com/robbiew/history/internal/wikimedia"
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
func DropFileData(path string) (string, string, string, string, string, string, string, string, string, string, string, error) {
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

	cleanPath := filepath.Clean(path)

	// Determine if the provided path is a file or directory.
	var filePath string
	if fi, err := os.Stat(cleanPath); err == nil && !fi.IsDir() {
		// Provided path is a file; use it directly.
		filePath = cleanPath
	} else {
		// Treat as directory: look for a case-insensitive "door32.sys"
		dirPath := cleanPath
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return "", "", "", "", "", "", "", "", "", "", "", fmt.Errorf("error reading directory %s: %v", dirPath, err)
		}
		found := ""
		for _, e := range entries {
			if strings.EqualFold(e.Name(), "door32.sys") {
				found = filepath.Join(dirPath, e.Name())
				break
			}
		}
		if found == "" {
			// As a fallback, also accept a direct filename appended (in case caller passed a directory-like string that didn't stat)
			possible := filepath.Join(dirPath, "door32.sys")
			if _, err := os.Stat(possible); err == nil {
				found = possible
			}
		}
		if found == "" {
			return "", "", "", "", "", "", "", "", "", "", "", fmt.Errorf("door32.sys not found in %s", dirPath)
		}
		filePath = found
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", "", "", fmt.Errorf("error opening %s: %v", filePath, err)
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
		return "", "", "", "", "", "", "", "", "", "", "", fmt.Errorf("scanner error: %v", err)
	}
	return commport, baudind, baudrate, bbsname, usernum, realname, username, seclevel, timeleft, emulation, node, nil
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
	// Correct ordinal suffix logic:
	// - 11, 12, 13 -> "th"
	// - otherwise, 1 -> "st", 2 -> "nd", 3 -> "rd", else "th"
	n := time.Now().Day()
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

// wrapText breaks text into lines that fit within maxWidth (rune-aware)
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		// Defensive: non-positive width -> return original text as single line
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
			// Start a new line
			if len(wr) <= maxWidth {
				current = append(current, wr...)
			} else {
				// Word longer than maxWidth -> truncate with ellipsis if possible
				if maxWidth > 3 {
					lines = append(lines, string(wr[:maxWidth-3])+"...")
				} else {
					lines = append(lines, string(wr[:maxWidth]))
				}
			}
			continue
		}
 
		// Attempt to add space + word
		if len(current)+1+len(wr) <= maxWidth {
			current = append(current, ' ')
			current = append(current, wr...)
		} else {
			// Flush current line and start new one with word (or truncated word)
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
 
// selectEventsByEra selects a small, varied set of events using an era-based strategy.
// It mirrors the era approach used in the JavaScript ENiGMA module: attempt to pick
// a small quota from each era, then fill remaining slots with random events.
func selectEventsByEra(allEvents []wikimedia.Event) []wikimedia.Event {
	if len(allEvents) == 0 {
		return nil
	}
 
	type eraDef struct {
		name       string
		min, max   int
		quota      int
	}
 
	eras := []eraDef{
		{name: "Ancient", min: 1, max: 500, quota: 1},
		{name: "Medieval", min: 501, max: 1500, quota: 1},
		{name: "Early Modern", min: 1501, max: 1800, quota: 1},
		{name: "Modern", min: 1801, max: 1950, quota: 1},
		{name: "Contemporary", min: 1951, max: 2030, quota: 1},
	}
 
	// Helper to create a unique key for an event
	keyFor := func(e wikimedia.Event) string {
		return fmt.Sprintf("%d|%s", e.Year, e.Text)
	}
 
	selected := make([]wikimedia.Event, 0, 5)
	seen := make(map[string]bool)
 
	// First pass: try to select quota from each era
	for _, era := range eras {
		// Collect eligible indices
		var eraEvents []int
		for i, ev := range allEvents {
			if ev.Year >= era.min && ev.Year <= era.max {
				eraEvents = append(eraEvents, i)
			}
		}
		if len(eraEvents) == 0 {
			continue
		}
		// Shuffle indices
		rand.Shuffle(len(eraEvents), func(i, j int) { eraEvents[i], eraEvents[j] = eraEvents[j], eraEvents[i] })
		// Pick up to quota
		for qi := 0; qi < era.quota && qi < len(eraEvents); qi++ {
			ev := allEvents[eraEvents[qi]]
			k := keyFor(ev)
			if !seen[k] {
				selected = append(selected, ev)
				seen[k] = true
			}
			if len(selected) >= 5 {
				break
			}
		}
		if len(selected) >= 5 {
			break
		}
	}
 
	// Fill remaining slots with random events if needed
	if len(selected) < 5 {
		// collect remaining indices not used
		var remaining []int
		for i, ev := range allEvents {
			if !seen[keyFor(ev)] {
				remaining = append(remaining, i)
			}
		}
		if len(remaining) > 0 {
			rand.Shuffle(len(remaining), func(i, j int) { remaining[i], remaining[j] = remaining[j], remaining[i] })
			need := 5 - len(selected)
			if need > len(remaining) {
				need = len(remaining)
			}
			for i := 0; i < need; i++ {
				selected = append(selected, allEvents[remaining[i]])
			}
		}
	}
 
	// Sort by year for stable display
	sort.SliceStable(selected, func(i, j int) bool { return selected[i].Year < selected[j].Year })
	return selected
}

func displayLoadingAnimation(done <-chan bool, wg *sync.WaitGroup) {
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
			if wg != nil {
				wg.Done()
			}
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

	// Retry strategy
	const maxAttempts = 3
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Use context with timeout for each attempt
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			cancel()
			return nil, err
		}

		req.Header.Set("User-Agent", "Go Day-in-History BBS Door/1.0 (github.com/robbiew/history)")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Encoding", "identity")

		client := &http.Client{
			// Let context handle timeouts; keep a reasonable transport timeout if desired.
			Timeout: 0,
		}

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			// Retry on transient network errors
			if attempt < maxAttempts {
				jitter := time.Duration(rand.Int63n(200))*time.Millisecond - 100*time.Millisecond
				time.Sleep(backoff + jitter)
				backoff *= 2
				continue
			}
			return nil, fmt.Errorf("network error: %v", err)
		}

		// Ensure body is closed for this attempt
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if readErr != nil {
			if attempt < maxAttempts {
				jitter := time.Duration(rand.Int63n(200))*time.Millisecond - 100*time.Millisecond
				time.Sleep(backoff + jitter)
				backoff *= 2
				continue
			}
			return nil, fmt.Errorf("failed to read response: %v", readErr)
		}

		// Accept HTTP 200. Retry on 429 or 5xx.
		if resp.StatusCode == http.StatusOK {
			var wikimediaResp WikimediaResponse
			if err := json.Unmarshal(body, &wikimediaResp); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %v", err)
			}

			var allEvents []WikimediaEvent
			for _, event := range wikimediaResp.Events {
				event.Type = "event"
				allEvents = append(allEvents, event)
			}
			// births/deaths intentionally excluded for a cleaner display

			// Shuffle deterministically seeded at startup
			if len(allEvents) > 1 {
				for i := len(allEvents) - 1; i > 0; i-- {
					j := rand.Intn(i + 1)
					allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
				}
			}
			return allEvents, nil
		}

		// Retryable statuses
		if resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 500 && resp.StatusCode < 600) {
			if attempt < maxAttempts {
				jitter := time.Duration(rand.Int63n(200))*time.Millisecond - 100*time.Millisecond
				time.Sleep(backoff + jitter)
				backoff *= 2
				continue
			}
			return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
		}

		// Non-retryable status
		return nil, fmt.Errorf("API returned status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("failed to fetch events after %d attempts", maxAttempts)
}

func generateEventList(termCfg terminal.TerminalConfig, wikiClient *wikimedia.Client, bypassCache, shuffle bool, strategy string) {
	// Start loading animation in background and fetch events concurrently
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go displayLoadingAnimation(done, &wg)
	
	// Determine month/day and fetch using provided client with a context timeout
	now := time.Now()
	monthStr := fmt.Sprintf("%02d", int(now.Month()))
	dayStr := fmt.Sprintf("%02d", now.Day())
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	events, err := wikiClient.FetchOnThisDay(ctx, monthStr, dayStr, bypassCache)
	cancel()
	
	// Stop the loading animation
	done <- true
	close(done)
	// Wait for the loader to finish clearing the line before continuing
	wg.Wait()
	
	// If fetching failed or no events, render an appropriate message using the existing quick path
	if err != nil {
		ClearScreen()
		MoveCursor(1, 8)
		fmt.Printf(RedHi+"Error fetching events: %v"+Reset+"\r\n", err)
		fmt.Print(WhiteHi+"Please check your internet connection and try again."+Reset+"\r\n")
		MoveCursor(1, 24)
		fmt.Print("                   " + BgBlueHi + WhiteHi + "<" + Reset + Cyan + "<  " + BlackHi + "... " + Reset + White + "press " + WhiteHi + "ANY KEY " + Reset + White + "to " + WhiteHi + "CONTINUE " + Reset + BlackHi + "... " + Reset + Cyan + ">" + BgBlue + WhiteHi + ">" + Reset)
		return
	}

	if len(events) == 0 {
		ClearScreen()
		MoveCursor(1, 8)
		fmt.Print(YellowHi + "No historical events found for today." + Reset + "\r\n")
		MoveCursor(1, 24)
		fmt.Print("                   " + BgBlueHi + WhiteHi + "<" + Reset + Cyan + "<  " + BlackHi + "... " + Reset + White + "press " + WhiteHi + "ANY KEY " + Reset + White + "to " + WhiteHi + "CONTINUE " + Reset + BlackHi + "... " + Reset + Cyan + ">" + BgBlue + WhiteHi + ">" + Reset)
		return
	}

	// If shuffle requested and strategy is oldest-first, treat it as random selection
	// so that -shuffle also randomizes which events are chosen (not just ordering).
	if shuffle && strategy == "oldest-first" {
		strategy = "random"
	}
	// Apply selection strategy (era-based, random, oldest-first)
	switch strategy {
	case "era-based":
		if sel := selectEventsByEra(events); len(sel) > 0 {
			events = sel
		}
	case "random":
		if len(events) > 1 {
			rand.Shuffle(len(events), func(i, j int) { events[i], events[j] = events[j], events[i] })
		}
		if len(events) > 5 {
			events = events[:5]
		}
	case "oldest-first":
		if len(events) > 1 {
			sort.SliceStable(events, func(i, j int) bool { return events[i].Year < events[j].Year })
		}
		if len(events) > 5 {
			events = events[:5]
		}
	// source-balanced strategy removed (not implemented)
	default:
		// Unknown strategy -> fallback to era-based
		if sel := selectEventsByEra(events); len(sel) > 0 {
			events = sel
		}
	}
	
	// If the global shuffle flag is set, randomize the order of the selected events
	if shuffle && len(events) > 1 {
		rand.Shuffle(len(events), func(i, j int) { events[i], events[j] = events[j], events[i] })
	}
	
	// Convert events to terminal-friendly types and render using the provided terminal config
	var tevents []terminal.Event
	for _, e := range events {
		tevents = append(tevents, terminal.Event{Year: e.Year, Text: e.Text})
	}

	terminal.RenderEvents(termCfg, tevents)
}

func main() {
	// Parse flags (moved from init)
	pathPtr := flag.String("path", "", "path to node directory")
	bypassCachePtr := flag.Bool("bypass-cache", false, "bypass cache and fetch fresh data")
	// Enable shuffle by default
	shufflePtr := flag.Bool("shuffle", true, "shuffle events every run (default: true)")
	strategyPtr := flag.String("strategy", "era-based", "selection strategy: era-based|random|oldest-first")
	cacheTTLS := flag.String("cache-ttl", "24h", "cache TTL (e.g., 1h, 30m)")
	flag.Parse()
	if *pathPtr == "" {
		fmt.Fprintf(os.Stderr, "missing path to node directory, e.g.: ./history -path /bbs/temp/1\n")
		os.Exit(2)
	}
	// Parse cache TTL
	cacheTTLDur, err := time.ParseDuration(*cacheTTLS)
	if err != nil {
		log.Printf("invalid cache-ttl '%s', defaulting to 24h: %v", *cacheTTLS, err)
		cacheTTLDur = 24 * time.Hour
	}

	// read the drop file and save to local struct
	commport, _, baudrate, bbsname, usernum, realname, username, seclevel, timeleft, emulation, node, err := DropFileData(*pathPtr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read dropfile: %v\n", err)
		os.Exit(1)
	}

	// convert some values to int (ignore conversion errors as before)
	intnode, _ := strconv.Atoi(node)
	intcommport, _ := strconv.Atoi(commport)
	intbaudrate, _ := strconv.Atoi(baudrate)
	intusernum, _ := strconv.Atoi(usernum)
	intseclevel, _ := strconv.Atoi(seclevel)
	inttimeleft, _ := strconv.Atoi(timeleft)
	intemulation, _ := strconv.Atoi(emulation)

	// detect terminal capabilities
	terminalName, loadableFonts, xtendPalette, cols, rows := DetectTerminalCapabilities()

	// local program state (no globals)
	localPd := Door32Drop{
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
		Terminal:      terminalName,
		LoadableFonts: loadableFonts,
		XtendPalette:  xtendPalette,
		Cols:          cols,
		Rows:          rows,
	}
	// Seed global PRNG for non-deterministic shuffling
	rand.Seed(time.Now().UnixNano())

	// Build terminal config
	termCfg := terminal.TerminalConfig{
		BbsName:  localPd.BbsName,
		UserName: localPd.UserName,
		RealName: localPd.RealName,
		Terminal: localPd.Terminal,
		Cols:     localPd.Cols,
		Rows:     localPd.Rows,
	}

	// Create wikimedia client (shared)
	wikiClient := wikimedia.NewClient("", cacheTTLDur)

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
		generateEventList(termCfg, wikiClient, *bypassCachePtr, *shufflePtr, *strategyPtr)
		_, err := tty.ReadRune()
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
}
