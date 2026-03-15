package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ── Screen buffer ────────────────────────────────────────────────────────────

var screenBuf strings.Builder

func scClear()          { screenBuf.Reset(); screenBuf.WriteString("\x1b[2J\x1b[H") }
func scHide()           { screenBuf.WriteString("\x1b[?25l") }
func scShow()           { screenBuf.WriteString("\x1b[?25h") }
func scMoveTo(r, c int) { fmt.Fprintf(&screenBuf, "\x1b[%d;%dH", r, c) }
func scClearLine()      { screenBuf.WriteString("\x1b[2K") }
func scWrite(s string)  { screenBuf.WriteString(s) }
func scFlush()          { os.Stdout.WriteString(screenBuf.String()) }

// ── Colors ───────────────────────────────────────────────────────────────────

const (
	cReset   = "\x1b[0m"
	cBold    = "\x1b[1m"
	cDim     = "\x1b[2m"
	cCorrect = "\x1b[32m"
	cWrong   = "\x1b[31m"
	cCursor  = "\x1b[7m"
	cPending = "\x1b[90m"
	cAccent  = "\x1b[36m"
	cWhite   = "\x1b[97m"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

func centerStr(s string, width int) string {
	pad := max((width - len(stripAnsi(s))) / 2, 0)
	return strings.Repeat(" ", pad) + s
}

// ── Word lists ───────────────────────────────────────────────────────────────

var wordLists = map[string][]string{
	"common": {
		"the", "be", "to", "of", "and", "a", "in", "that", "have", "it", "for", "not", "on",
		"with", "he", "as", "you", "do", "at", "this", "but", "his", "by", "from", "they",
		"we", "say", "her", "she", "or", "an", "will", "my", "one", "all", "would", "there",
		"their", "what", "so", "up", "out", "if", "about", "who", "get", "which", "go", "me",
		"when", "make", "can", "like", "time", "no", "just", "him", "know", "take", "people",
		"into", "year", "your", "good", "some", "could", "them", "see", "other", "than",
		"then", "now", "look", "only", "come", "its", "over", "think", "also", "back", "after",
		"use", "two", "how", "our", "work", "first", "well", "way", "even", "new", "want",
		"because", "any", "these", "give", "day", "most", "us", "between", "need", "large",
		"often", "hand", "high", "place", "hold", "turn", "where", "much", "before", "move",
		"right", "boy", "old", "too", "same", "tell", "does", "set", "three", "small", "put",
		"end", "does", "another", "well", "large", "big", "down", "never", "start", "city",
		"play", "small", "number", "off", "always", "next", "open", "seem", "together",
		"white", "children", "begin", "got", "walk", "example", "ease", "paper", "group",
		"always", "music", "those", "both", "mark", "book", "letter", "until", "mile",
		"river", "car", "feet", "care", "second", "enough", "plain", "girl", "usual",
		"young", "ready", "above", "ever", "red", "list", "though", "feel", "talk", "bird",
		"soon", "body", "dog", "family", "direct", "pose", "leave", "song", "measure",
		"door", "product", "black", "short", "numeral", "class", "wind", "question",
		"happen", "complete", "ship", "area", "half", "rock", "order", "fire", "south",
		"problem", "piece", "told", "knew", "pass", "since", "top", "whole", "king",
		"space", "heard", "best", "hour", "better", "during", "hundred", "five", "remember",
		"step", "early", "hold", "west", "ground", "interest", "reach", "fast", "five",
	},
	"code": {
		"function", "return", "const", "let", "var", "if", "else", "for", "while", "do",
		"switch", "case", "break", "continue", "class", "extends", "import", "export",
		"default", "new", "this", "null", "undefined", "true", "false", "typeof", "void",
		"delete", "throw", "try", "catch", "finally", "async", "await", "yield", "static",
		"public", "private", "protected", "abstract", "interface", "type", "enum", "from",
		"string", "number", "boolean", "object", "array", "promise", "callback", "event",
		"module", "require", "console", "log", "error", "warn", "info", "debug", "process",
		"window", "document", "element", "query", "selector", "style", "class", "id",
		"props", "state", "render", "component", "hook", "effect", "ref", "context",
		"reduce", "filter", "map", "forEach", "find", "some", "every", "includes",
		"push", "pop", "shift", "unshift", "splice", "slice", "concat", "join", "split",
		"length", "keys", "values", "entries", "assign", "spread", "rest", "destructure",
		"arrow", "template", "literal", "optional", "chaining", "nullish", "coalescing",
	},
	"quotes": {
		"the quick brown fox jumps over the lazy dog",
		"to be or not to be that is the question",
		"all that glitters is not gold",
		"it always seems impossible until it is done",
		"the only way to do great work is to love what you do",
		"in the middle of difficulty lies opportunity",
		"life is what happens when you are busy making other plans",
		"the future belongs to those who believe in the beauty of their dreams",
		"it does not matter how slowly you go as long as you do not stop",
		"success is not final failure is not fatal it is the courage to continue that counts",
	},
}

// ── Config ───────────────────────────────────────────────────────────────────

type config struct {
	WordCount int    `json:"wordCount"`
	Mode      string `json:"mode"`
	TimeLimit int    `json:"timeLimit"`
	ErrorMode string `json:"errorMode"`
}

var cfg = config{
	WordCount: 25,
	Mode:      "common",
	TimeLimit: 0,
	ErrorMode: "normal",
}

func settingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ttyper.json")
}

func loadSettings() {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return
	}
	var saved config
	if json.Unmarshal(data, &saved) == nil {
		if saved.WordCount > 0 {
			cfg.WordCount = saved.WordCount
		}
		if saved.Mode != "" {
			cfg.Mode = saved.Mode
		}
		cfg.TimeLimit = saved.TimeLimit
		if saved.ErrorMode != "" {
			cfg.ErrorMode = saved.ErrorMode
		}
	}
}

func saveSettings() {
	data, _ := json.Marshal(cfg)
	os.WriteFile(settingsPath(), data, 0644)
}

// ── High scores ──────────────────────────────────────────────────────────────

type highScore struct {
	WPM int `json:"wpm"`
	Acc int `json:"acc"`
}

var highScores map[string]highScore

func scoresPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ttyper-scores.json")
}

func scoreKey() string {
	if cfg.TimeLimit > 0 {
		return fmt.Sprintf("%s|%s|t%d", cfg.Mode, cfg.ErrorMode, cfg.TimeLimit)
	}
	return fmt.Sprintf("%s|%s|w%d", cfg.Mode, cfg.ErrorMode, cfg.WordCount)
}

func loadScores() {
	highScores = make(map[string]highScore)
	data, err := os.ReadFile(scoresPath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &highScores)
}

func saveScore(wpm, acc int) bool {
	key := scoreKey()
	prev, exists := highScores[key]
	if exists && wpm <= prev.WPM {
		return false
	}
	highScores[key] = highScore{WPM: wpm, Acc: acc}
	data, _ := json.MarshalIndent(highScores, "", "  ")
	os.WriteFile(scoresPath(), data, 0644)
	return true
}

func getHighScore() (highScore, bool) {
	s, ok := highScores[scoreKey()]
	return s, ok
}

// scoreKeyWith builds a score key as if a specific menu item were selected.
func scoreKeyWith(item menuItem) string {
	mode := cfg.Mode
	errorMode := cfg.ErrorMode
	wordCount := cfg.WordCount
	timeLimit := cfg.TimeLimit

	switch item.category {
	case "mode":
		mode = item.mode
	case "errors":
		errorMode = item.mode
	case "words":
		wordCount = item.value
		timeLimit = 0
	case "time":
		timeLimit = item.value
	}

	if timeLimit > 0 {
		return fmt.Sprintf("%s|%s|t%d", mode, errorMode, timeLimit)
	}
	return fmt.Sprintf("%s|%s|w%d", mode, errorMode, wordCount)
}

func getHighScoreFor(item menuItem) (highScore, bool) {
	s, ok := highScores[scoreKeyWith(item)]
	return s, ok
}

// ── Menu items ───────────────────────────────────────────────────────────────

type menuItem struct {
	category string // "mode", "words", "time", "errors"
	mode     string
	value    int
}

func (m menuItem) isActive() bool {
	switch m.category {
	case "mode":
		return cfg.Mode == m.mode
	case "words":
		return cfg.WordCount == m.value && cfg.TimeLimit == 0
	case "time":
		return cfg.TimeLimit == m.value
	case "errors":
		return cfg.ErrorMode == m.mode
	}
	return false
}

func (m menuItem) label() string {
	switch m.category {
	case "mode":
		return m.mode
	case "errors":
		return m.mode
	case "time":
		if m.value == 0 {
			return "off"
		}
		return strconv.Itoa(m.value) + "s"
	}
	return strconv.Itoa(m.value)
}

var allMenuItems = []menuItem{
	{category: "mode", mode: "common"},
	{category: "mode", mode: "code"},
	{category: "mode", mode: "quotes"},
	{category: "errors", mode: "normal"},
	{category: "errors", mode: "strict"},
	{category: "errors", mode: "impossible"},
	{category: "words", value: 10},
	{category: "words", value: 25},
	{category: "words", value: 50},
	{category: "words", value: 100},
	{category: "time", value: 0},
	{category: "time", value: 15},
	{category: "time", value: 30},
	{category: "time", value: 60},
}

// ── Game state ───────────────────────────────────────────────────────────────

type gameState struct {
	words             []string
	typed             []string
	done              []bool
	currentWord       int
	currentInput      string
	startTime         time.Time
	endTime           time.Time
	started           bool
	finished          bool
	failed            bool
	newBest           bool
	totalKeystrokes   int
	correctKeystrokes int
	mode              string
	timeLimit         int
	timeLeft          int
	ticker            *time.Ticker
	showMenu          bool
	menuCursor        int
	quoteQueue        []int
}

var state gameState

// ── Layout ───────────────────────────────────────────────────────────────────

const (
	rowHeader = 2
	rowWords  = 5
	rowStats  = 11
)

// ── Word list building ───────────────────────────────────────────────────────

func buildWordList(mode string, count int) []string {
	list := wordLists[mode]
	if mode == "quotes" {
		q := list[rand.Intn(len(list))]
		return strings.Split(q, " ")
	}
	shuffled := make([]string, len(list))
	copy(shuffled, list)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	var words []string
	for len(words) < count {
		words = append(words, shuffled...)
	}
	return words[:count]
}

func nextQuoteWords() []string {
	if len(state.quoteQueue) == 0 {
		quotes := wordLists["quotes"]
		state.quoteQueue = make([]int, len(quotes))
		for i := range state.quoteQueue {
			state.quoteQueue[i] = i
		}
		rand.Shuffle(len(state.quoteQueue), func(i, j int) {
			state.quoteQueue[i], state.quoteQueue[j] = state.quoteQueue[j], state.quoteQueue[i]
		})
	}
	idx := state.quoteQueue[len(state.quoteQueue)-1]
	state.quoteQueue = state.quoteQueue[:len(state.quoteQueue)-1]
	return strings.Split(wordLists["quotes"][idx], " ")
}

func ensureWordBuffer() {
	const buffer = 30
	remaining := len(state.words) - state.currentWord
	if remaining >= buffer {
		return
	}

	if cfg.Mode == "quotes" {
		for len(state.words)-state.currentWord < buffer {
			words := nextQuoteWords()
			state.words = append(state.words, words...)
			state.typed = append(state.typed, make([]string, len(words))...)
			state.done = append(state.done, make([]bool, len(words))...)
		}
	} else {
		need := buffer - remaining
		extra := buildWordList(cfg.Mode, need)
		state.words = append(state.words, extra...)
		state.typed = append(state.typed, make([]string, len(extra))...)
		state.done = append(state.done, make([]bool, len(extra))...)
	}
}

// ── State init ───────────────────────────────────────────────────────────────

func initState() {
	if state.ticker != nil {
		state.ticker.Stop()
	}
	words := buildWordList(cfg.Mode, cfg.WordCount)
	state = gameState{
		words:    words,
		typed:    make([]string, len(words)),
		done:     make([]bool, len(words)),
		mode:     cfg.Mode,
		timeLimit: cfg.TimeLimit,
		timeLeft: cfg.TimeLimit,
	}
	if cfg.TimeLimit > 0 {
		ensureWordBuffer()
	}
}

// ── Calculations ─────────────────────────────────────────────────────────────

func calcWPM() int {
	if state.startTime.IsZero() {
		return 0
	}
	end := state.endTime
	if end.IsZero() {
		end = time.Now()
	}
	elapsed := end.Sub(state.startTime).Minutes()
	if elapsed == 0 {
		return 0
	}
	chars := 0
	for _, t := range state.typed {
		chars += len(t)
	}
	return int(float64(chars) / 5.0 / elapsed)
}

func calcAccuracy() int {
	if state.totalKeystrokes == 0 {
		return 100
	}
	return int(float64(state.correctKeystrokes) / float64(state.totalKeystrokes) * 100)
}

// ── Rendering ────────────────────────────────────────────────────────────────

func render() {
	w, _ := getTermSize()

	if state.showMenu {
		renderMenu(w)
		return
	}
	if state.finished {
		renderResults(w)
		return
	}
	renderGame(w)
}

func renderGame(w int) {
	scClear()
	scHide()

	// Header
	scMoveTo(rowHeader, 1)
	scClearLine()
	modeLabel := state.mode
	if modeLabel == "common" {
		modeLabel = "words"
	}
	var timePart string
	if state.timeLimit > 0 {
		timePart = fmt.Sprintf(" · %s%ds%s", cAccent, state.timeLeft, cReset)
	} else {
		timePart = fmt.Sprintf(" · %s%d words%s", cAccent, cfg.WordCount, cReset)
	}
	var bestPart string
	if best, ok := getHighScore(); ok {
		bestPart = fmt.Sprintf(" · %sbest: %d wpm%s", cDim, best.WPM, cReset)
	}
	header := fmt.Sprintf("%s%sttyper%s  %s%s%s%s", cBold, cAccent, cReset, cDim, modeLabel, timePart, bestPart)
	scWrite(centerStr(header, w))

	// Live stats
	scMoveTo(rowHeader+1, 1)
	scClearLine()
	if state.started {
		wpm := calcWPM()
		acc := calcAccuracy()
		live := fmt.Sprintf("%swpm: %s%d%s  %sacc: %s%d%%%s", cDim, cWhite, wpm, cReset, cDim, cWhite, acc, cReset)
		scWrite(centerStr(live, w))
	} else {
		scWrite(centerStr(cDim+"start typing to begin..."+cReset, w))
	}

	// Word display
	renderWordDisplay(w)

	// Hint
	scMoveTo(rowStats, 1)
	scClearLine()
	scWrite(centerStr(cDim+"tab: restart  ctrl+c: quit  ctrl+o: menu"+cReset, w))

	scFlush()
}

func renderWordDisplay(w int) {
	maxWidth := min(w - 4, 70)
	margin := max((w - maxWidth) / 2, 0)
	gap := "  "

	// Group word indices into lines
	var lines [][]int
	var curLine []int
	lineLen := 0

	for i := 0; i < len(state.words); i++ {
		word := state.words[i]
		needed := len(word)
		if len(curLine) > 0 {
			needed += len(gap)
		}
		if lineLen+needed > maxWidth && len(curLine) > 0 {
			lines = append(lines, curLine)
			curLine = nil
			lineLen = 0
			needed = len(word)
		}
		curLine = append(curLine, i)
		lineLen += needed
	}
	if len(curLine) > 0 {
		lines = append(lines, curLine)
	}

	// Find line containing current word
	currentLine := 0
	for li, ln := range lines {
		for _, idx := range ln {
			if idx == state.currentWord {
				currentLine = li
			}
		}
	}

	// Display 3 lines centered on current
	display := [3]int{currentLine - 1, currentLine, currentLine + 1}
	for di, li := range display {
		row := rowWords + di
		scMoveTo(row, 1)
		scClearLine()

		if li < 0 || li >= len(lines) {
			continue
		}

		var out strings.Builder
		for j, idx := range lines[li] {
			word := state.words[idx]
			if j > 0 {
				out.WriteString(cDim + gap + cReset)
			}

			if state.done[idx] {
				typed := state.typed[idx]
				maxLen := max(len(typed), len(word))
				for ci := range maxLen {
					if ci >= len(word) {
						out.WriteString(cWrong + string(typed[ci]) + cReset)
					} else if ci >= len(typed) {
						out.WriteString(cWrong + string(word[ci]) + cReset)
					} else if typed[ci] == word[ci] {
						out.WriteString(cCorrect + string(word[ci]) + cReset)
					} else {
						out.WriteString(cWrong + string(word[ci]) + cReset)
					}
				}
			} else if idx == state.currentWord {
				typed := state.currentInput
				for ci := 0; ci < len(word); ci++ {
					if ci < len(typed) {
						color := cCorrect
						if typed[ci] != word[ci] {
							color = cWrong
						}
						out.WriteString(color + string(word[ci]) + cReset)
					} else if ci == len(typed) {
						out.WriteString(cCursor + string(word[ci]) + cReset)
					} else {
						out.WriteString(cPending + string(word[ci]) + cReset)
					}
				}
				for ci := len(word); ci < len(typed); ci++ {
					out.WriteString(cWrong + string(typed[ci]) + cReset)
				}
			} else {
				out.WriteString(cPending + word + cReset)
			}
		}

		scWrite(strings.Repeat(" ", margin) + out.String())
	}
}

func renderResults(w int) {
	_, h := getTermSize()
	scClear()
	scShow()

	wpm := calcWPM()
	acc := calcAccuracy()
	elapsed := state.endTime.Sub(state.startTime).Seconds()

	correct := 0
	wrong := 0
	for i := range state.words {
		if state.done[i] {
			if state.typed[i] == state.words[i] {
				correct++
			} else {
				wrong++
			}
		}
	}

	var title string
	if state.failed {
		title = cBold + cWrong + "-- failed --" + cReset
	} else {
		title = cBold + cAccent + "-- results --" + cReset
	}

	lines := []string{
		"",
		title,
		"",
		fmt.Sprintf("%swpm     %s%s%d%s", cDim, cBold, cWhite, wpm, cReset),
		fmt.Sprintf("%sacc     %s%s%d%%%s", cDim, cBold, cWhite, acc, cReset),
		fmt.Sprintf("%stime    %s%s%.1fs%s", cDim, cBold, cWhite, elapsed, cReset),
		fmt.Sprintf("%scorrect %s%s%d%s", cDim, cBold, cCorrect, correct, cReset),
		fmt.Sprintf("%swrong   %s%s%d%s", cDim, cBold, cWrong, wrong, cReset),
	}

	if state.newBest {
		lines = append(lines, "", cBold+cCorrect+"new best!"+cReset)
	} else if best, ok := getHighScore(); ok {
		lines = append(lines, "", fmt.Sprintf("%sbest    %s%s%d wpm%s", cDim, cBold, cAccent, best.WPM, cReset))
	}

	lines = append(lines, "", cDim+"tab: restart  ctrl+o: menu  ctrl+c: quit"+cReset)

	startRow := h/2 - len(lines)/2
	for i, l := range lines {
		scMoveTo(startRow+i, 1)
		scWrite(centerStr(l, w))
	}
	scFlush()
}

func renderMenu(w int) {
	_, h := getTermSize()
	scClear()
	scShow()

	idx := 0
	itemLine := func(item menuItem, desc string) string {
		i := idx
		idx++
		active := item.isActive()
		isCursor := state.menuCursor == i

		var prefix, text string
		if isCursor {
			prefix = cAccent + "▸ "
			text = cWhite + item.label()
		} else if active {
			prefix = cCorrect + "· "
			text = cCorrect + item.label()
		} else {
			prefix = "  "
			text = cPending + item.label()
		}
		line := "  " + prefix + text + cReset
		if desc != "" {
			line += "  " + cDim + desc + cReset
		}
		if s, ok := getHighScoreFor(item); ok {
			line += "  " + cDim + fmt.Sprintf("%d wpm", s.WPM) + cReset
		}
		return line
	}

	lines := []string{
		cBold + cAccent + "-- settings --" + cReset,
		"",
		cDim + "mode" + cReset,
		itemLine(allMenuItems[0], ""),
		itemLine(allMenuItems[1], ""),
		itemLine(allMenuItems[2], ""),
		"",
		cDim + "difficulty" + cReset,
		itemLine(allMenuItems[3], "mistakes are allowed"),
		itemLine(allMenuItems[4], "submit a wrong word and you fail"),
		itemLine(allMenuItems[5], "one wrong key and you fail"),
		"",
		cDim + "words" + cReset + "  " + cDim + "(word count mode)" + cReset,
		itemLine(allMenuItems[6], ""),
		itemLine(allMenuItems[7], ""),
		itemLine(allMenuItems[8], ""),
		itemLine(allMenuItems[9], ""),
		"",
		cDim + "time" + cReset + "   " + cDim + "(timed mode)" + cReset,
		itemLine(allMenuItems[10], ""),
		itemLine(allMenuItems[11], ""),
		itemLine(allMenuItems[12], ""),
		itemLine(allMenuItems[13], ""),
		"",
		cDim + "↑↓/jk: navigate  enter: select  esc: close  tab: restart" + cReset,
	}

	startRow := h/2 - len(lines)/2
	for i, l := range lines {
		scMoveTo(startRow+i, 1)
		scWrite(centerStr(l, w))
	}
	scFlush()
}

// ── Key reading ──────────────────────────────────────────────────────────────

func readKey() string {
	b := make([]byte, 16)
	n, err := os.Stdin.Read(b)
	if err != nil || n == 0 {
		return ""
	}

	if b[0] == 0x1b {
		if n == 1 {
			return "\x1b"
		}
		if n >= 3 && b[1] == '[' {
			return string(b[:3])
		}
		return "\x1b"
	}

	return string(b[0])
}

// ── Input handling ───────────────────────────────────────────────────────────

var (
	keys     chan string
	tickChan <-chan time.Time
	restore  func()
)

func handleKey(key string) bool {
	// Ctrl+C
	if key == "\x03" {
		return true
	}

	// Tab = restart
	if key == "\x09" {
		initState()
		render()
		return false
	}

	// Ctrl+O = menu toggle
	if key == "\x0f" {
		state.showMenu = !state.showMenu
		render()
		return false
	}

	// Esc = close menu
	if key == "\x1b" {
		if state.showMenu {
			state.showMenu = false
			render()
		}
		return false
	}

	if state.showMenu {
		handleMenuKey(key)
		return false
	}

	if state.finished {
		return false
	}

	// Start timer on first keystroke
	if !state.started {
		state.started = true
		state.startTime = time.Now()
		if state.timeLimit > 0 {
			state.timeLeft = state.timeLimit
			state.ticker = time.NewTicker(time.Second)
			tickChan = state.ticker.C
		}
	}

	word := state.words[state.currentWord]

	// Backspace
	if key == "\x7f" || key == "\x08" {
		if len(state.currentInput) > 0 {
			state.currentInput = state.currentInput[:len(state.currentInput)-1]
		} else if state.currentWord > 0 {
			state.currentWord--
			state.currentInput = state.typed[state.currentWord]
			state.done[state.currentWord] = false
			state.typed[state.currentWord] = ""
		}
		render()
		return false
	}

	// Space = submit word
	if key == " " {
		if len(state.currentInput) == 0 {
			return false
		}
		// Strict: moving on from a word with a mistake = instant fail
		if cfg.ErrorMode == "strict" && state.currentInput != word {
			state.failed = true
			finishGame()
			return false
		}
		submitWord()
		return false
	}

	// Printable ASCII
	if len(key) == 1 && key[0] >= ' ' {
		state.totalKeystrokes++
		correct := len(state.currentInput) < len(word) && key[0] == word[len(state.currentInput)]
		if correct {
			state.correctKeystrokes++
		}

		// Impossible: any wrong key = instant fail
		if cfg.ErrorMode == "impossible" && !correct {
			state.failed = true
			finishGame()
			return false
		}

		state.currentInput += key
		render()
		return false
	}

	return false
}

func handleMenuKey(key string) {
	total := len(allMenuItems)

	// Up: arrow up or k
	if key == "\x1b[A" || key == "k" {
		state.menuCursor = (state.menuCursor - 1 + total) % total
		render()
		return
	}

	// Down: arrow down or j
	if key == "\x1b[B" || key == "j" {
		state.menuCursor = (state.menuCursor + 1) % total
		render()
		return
	}

	// Enter
	if key == "\r" || key == "\n" {
		item := allMenuItems[state.menuCursor]
		switch item.category {
		case "mode":
			cfg.Mode = item.mode
		case "errors":
			cfg.ErrorMode = item.mode
		case "words":
			cfg.WordCount = item.value
			cfg.TimeLimit = 0
		case "time":
			cfg.TimeLimit = item.value
		}
		saveSettings()
		render()
		return
	}
}

func submitWord() {
	i := state.currentWord
	state.typed[i] = state.currentInput
	state.done[i] = true
	state.currentInput = ""
	state.currentWord++

	if cfg.TimeLimit == 0 && state.currentWord >= len(state.words) {
		finishGame()
		return
	}

	if cfg.TimeLimit > 0 {
		ensureWordBuffer()
	}

	render()
}

func finishGame() {
	state.endTime = time.Now()
	state.finished = true
	if !state.failed {
		state.newBest = saveScore(calcWPM(), calcAccuracy())
	}
	if state.ticker != nil {
		state.ticker.Stop()
		state.ticker = nil
		tickChan = nil
	}
	render()
}

// ── Cleanup ──────────────────────────────────────────────────────────────────

func cleanup() {
	if state.ticker != nil {
		state.ticker.Stop()
	}
	os.Stdout.WriteString("\x1b[?25h\n")
	if restore != nil {
		restore()
	}
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	loadSettings()
	loadScores()

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-m", "--mode":
			if i+1 < len(args) {
				if _, ok := wordLists[args[i+1]]; ok {
					cfg.Mode = args[i+1]
				}
				i++
			}
		case "-n", "--words":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
					cfg.WordCount = n
				}
				i++
			}
		case "-t", "--time":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
					cfg.TimeLimit = n
				}
				i++
			}
		case "-h", "--help":
			fmt.Println(`ttyper - terminal typing speed test

Usage: ttyper [options]

  -m, --mode   <mode>   Word list: common, code, quotes  (default: common)
  -n, --words  <n>      Number of words  (default: 25)
  -t, --time   <secs>   Timed mode in seconds (overrides -n)
  -h, --help            Show this help

Controls:
  space         submit word
  backspace     delete char
  tab           restart
  ctrl+o        open settings menu
  ctrl+c        quit`)
			os.Exit(0)
		}
	}

	stat, _ := os.Stdin.Stat()
	if stat.Mode()&os.ModeCharDevice == 0 {
		fmt.Fprintln(os.Stderr, "ttyper requires an interactive terminal (TTY).")
		os.Exit(1)
	}

	var err error
	restore, err = enableRawMode()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to enable raw mode:", err)
		os.Exit(1)
	}
	defer cleanup()

	initState()

	keys = make(chan string, 1)
	go func() {
		for {
			k := readKey()
			if k != "" {
				keys <- k
			}
		}
	}()

	render()

	for {
		select {
		case key := <-keys:
			if handleKey(key) {
				return
			}
		case <-tickChanFunc():
			state.timeLeft--
			if state.timeLeft <= 0 {
				finishGame()
			} else {
				render()
			}
		}
	}
}

func tickChanFunc() <-chan time.Time {
	return tickChan
}
