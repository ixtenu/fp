package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// ---- configuration ----------------------------------------------------------

type config struct {
	maxWidth       int
	sentenceSpaces int
	aesthetic      bool
}

var abbreviations = map[string]bool{
	"Mr.": true, "Ms.": true, "Mrs.": true, "Dr.": true,
	"Prof.": true, "St.": true, "vs.": true, "approx.": true,
	"fig.": true, "Fig.": true, "vol.": true, "Vol.": true,
	"e.g.": true, "i.e.": true, "viz.": true,
}

// prefixTokens in longest-first order so longest match wins.
var prefixTokens = []string{"///", "//", "--", "##", "rem", ">", "#", "%", "!", "*"}

var numberedRe = regexp.MustCompile(`^[0-9]+\. `)

func parseConfig() config {
	// 1. Hard-coded defaults.
	cfg := config{
		maxWidth:       80,
		sentenceSpaces: 2,
		aesthetic:      false,
	}

	// 2. INI file (~/.config/fp/fp.ini).
	loadIni(&cfg)

	// 3. Project INI file (.fp.ini, searched upward from CWD).
	loadProjectIni(&cfg)

	// 4. Environment variables.
	if v := os.Getenv("FP_LINE_WIDTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.maxWidth = n
		}
	}
	if v := os.Getenv("FP_SENTENCE_SPACES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.sentenceSpaces = n
		}
	}
	if os.Getenv("FP_AESTHETIC_WRAP") != "" {
		cfg.aesthetic = true
	}

	// 5. Command-line flags.
	// Pre-process args: expand -wN and -sN into -w=N and -s=N so that the
	// standard flag package can parse them.
	args := os.Args[1:]
	expanded := make([]string, 0, len(args))
	for _, a := range args {
		if len(a) > 2 && a[0] == '-' && (a[1] == 'w' || a[1] == 's') {
			expanded = append(expanded, "-"+string(a[1])+"="+a[2:])
		} else {
			expanded = append(expanded, a)
		}
	}
	fs := flag.NewFlagSet("fp", flag.ExitOnError)
	w := fs.Int("w", cfg.maxWidth, "max line width")
	s := fs.Int("s", cfg.sentenceSpaces, "spaces after sentence end")
	a := fs.Bool("a", cfg.aesthetic, "aesthetic wrap")
	fs.Parse(expanded) //nolint:errcheck // ExitOnError handles it
	cfg.maxWidth = *w
	cfg.sentenceSpaces = *s
	cfg.aesthetic = *a
	return cfg
}

// loadIni reads $XDG_CONFIG_HOME/fp/fp.ini (or ~/.config/fp/fp.ini if
// XDG_CONFIG_HOME is unset) and applies any recognised settings to cfg.
func loadIni(cfg *config) {
	var configDir string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = xdg
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		configDir = filepath.Join(home, ".config")
	}
	parseIniFile(filepath.Join(configDir, "fp", "fp.ini"), cfg)
}

// loadProjectIni walks up from the current working directory looking for a
// .fp.ini file and applies the first one found to cfg.
func loadProjectIni(cfg *config) {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		path := filepath.Join(dir, ".fp.ini")
		if _, err := os.Stat(path); err == nil {
			parseIniFile(path, cfg)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return // reached filesystem root
		}
		dir = parent
	}
}

// parseIniFile reads an INI file at path and applies recognised settings to
// cfg.  If the file does not exist, is unreadable, or contains invalid content,
// cfg is left unchanged.
func parseIniFile(path string, cfg *config) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	tmp := *cfg // parse into a copy; only commit if the whole file is valid
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || line[0] == '#' || line[0] == ';' || line[0] == '[' {
			continue // blank lines, comments, section headers: skip
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return // not a key=value line: file is invalid, abort
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		switch key {
		case "line_width":
			n, err := strconv.Atoi(val)
			if err != nil {
				return
			}
			tmp.maxWidth = n
		case "sentence_spaces":
			n, err := strconv.Atoi(val)
			if err != nil {
				return
			}
			tmp.sentenceSpaces = n
		case "aesthetic_wrap":
			switch strings.ToLower(val) {
			case "true", "1", "yes", "on":
				tmp.aesthetic = true
			case "false", "0", "no", "off":
				tmp.aesthetic = false
			default:
				return
			}
		default:
			// Unknown keys are silently ignored (forward-compatible).
			break
		}
	}
	*cfg = tmp
}

// ---- column width -----------------------------------------------------------

func colWidth(s string) int {
	return runewidth.StringWidth(s)
}

// ---- prefix detection -------------------------------------------------------

// detectPrefix returns (prefixWithTrailingSpace, restOfLine).
// If no prefix is found both values equal ("", line).
func detectPrefix(line string) (string, string) {
	for _, tok := range prefixTokens {
		var candidate string
		if strings.EqualFold(tok, "rem") {
			if len(line) >= 4 && strings.EqualFold(line[:3], "rem") && line[3] == ' ' {
				candidate = line[:4]
			}
		} else {
			candidate = tok + " "
		}
		if candidate != "" && strings.HasPrefix(line, candidate) {
			return candidate, line[len(candidate):]
		}
	}
	return "", line
}

// stripPrefix removes the expected prefix from line.
// Returns (stripped, true) if the prefix was present, or ("", false) if not.
func stripPrefix(line, prefix string) (string, bool) {
	if prefix == "" {
		return line, true
	}
	// Also accept the bare prefix token (without trailing space) for paragraph breaks.
	tok := strings.TrimRight(prefix, " ")
	if strings.TrimRight(line, " \t") == tok {
		return "", true // blank prefixed line
	}
	if strings.HasPrefix(line, prefix) {
		return line[len(prefix):], true
	}
	return line, false
}

// ---- list token detection ---------------------------------------------------

// detectListToken returns (token+space, rest) or ("", line).
func detectListToken(s string) (string, string) {
	for _, tok := range []string{"- ", "+ ", "#. "} {
		if strings.HasPrefix(s, tok) {
			return tok, s[len(tok):]
		}
	}
	if loc := numberedRe.FindStringIndex(s); loc != nil {
		tok := s[:loc[1]]
		return tok, s[loc[1]:]
	}
	return "", s
}

// ---- word / sentence handling -----------------------------------------------

type word struct {
	text          string
	width         int
	sentenceBreak bool // true if a sentence ends after this word
}

// closeQuote returns true if r is a closing quotation mark.
func closeQuote(r rune) bool {
	return r == '"' || r == '\'' || r == '\u201D' || r == '\u2019'
}

// sentenceEndChar returns true if r ends a sentence (.?!).
func sentenceEndChar(r rune) bool {
	return r == '.' || r == '?' || r == '!'
}

// wordEndsSentence decides whether this word ends a sentence.
func wordEndsSentence(w string) bool {
	if len(w) == 0 {
		return false
	}
	// Strip trailing close quotes to inspect the sentence-end char.
	base := w
	for len(base) > 0 {
		r, size := lastRune(base)
		if closeQuote(r) {
			base = base[:len(base)-size]
		} else {
			break
		}
	}
	if len(base) == 0 {
		return false
	}
	r, _ := lastRune(base)
	if !sentenceEndChar(r) {
		return false
	}
	// Check abbreviations (case-sensitive for abbreviations).
	if abbreviations[base] || abbreviations[w] {
		return false
	}
	return true
}

func lastRune(s string) (rune, int) {
	r, size := utf8.DecodeLastRuneInString(s)
	return r, size
}

// extractWords splits text into words, preserving sentence-break information.
func extractWords(text string, cfg config) []word {
	raw := strings.Fields(text)
	words := make([]word, len(raw))
	for i, w := range raw {
		words[i] = word{
			text:          w,
			width:         colWidth(w),
			sentenceBreak: cfg.sentenceSpaces > 1 && wordEndsSentence(w),
		}
	}
	return words
}

// spaceAfter returns the number of spaces to insert after words[i].
func spaceAfter(words []word, i, sentenceSpaces int) int {
	if words[i].sentenceBreak {
		return sentenceSpaces
	}
	return 1
}

// ---- line wrapping ----------------------------------------------------------

func joinWords(words []word, start, end, sentenceSpaces int) string {
	if start >= end {
		return ""
	}
	var b strings.Builder
	b.WriteString(words[start].text)
	for i := start + 1; i < end; i++ {
		sp := spaceAfter(words, i-1, sentenceSpaces)
		for j := 0; j < sp; j++ {
			b.WriteByte(' ')
		}
		b.WriteString(words[i].text)
	}
	return b.String()
}

// wrapGreedy returns line break indices: each element is the start index of a line.
func wrapGreedy(words []word, maxWidth, sentenceSpaces int) []int {
	if len(words) == 0 {
		return nil
	}
	starts := []int{0}
	lineW := words[0].width
	for i := 1; i < len(words); i++ {
		sp := spaceAfter(words, i-1, sentenceSpaces)
		needed := lineW + sp + words[i].width
		if needed > maxWidth {
			starts = append(starts, i)
			lineW = words[i].width
		} else {
			lineW = needed
		}
	}
	return starts
}

// wrapKnuthPlass returns line-start indices using the Knuth-Plass algorithm
// (without hyphenation). Badness = (slack)^3 per line; last line free.
func wrapKnuthPlass(words []word, maxWidth, sentenceSpaces int) []int {
	n := len(words)
	if n == 0 {
		return nil
	}
	const inf = math.MaxFloat64 / 2
	opt := make([]float64, n+1)
	from := make([]int, n+1)
	for i := range opt {
		opt[i] = inf
	}
	opt[0] = 0

	for i := 1; i <= n; i++ {
		w := 0
		for j := i - 1; j >= 0; j-- {
			if j < i-1 {
				w += spaceAfter(words, j, sentenceSpaces)
			}
			w += words[j].width
			if w > maxWidth {
				break
			}
			if opt[j] >= inf {
				continue
			}
			var pen float64
			if i == n {
				pen = 0 // no penalty for last line
			} else {
				slack := float64(maxWidth - w)
				pen = slack * slack * slack
			}
			if total := opt[j] + pen; total < opt[i] {
				opt[i] = total
				from[i] = j
			}
		}
		// Fallback: if no fit found, force a break (one word per line).
		if opt[i] >= inf {
			opt[i] = inf - 1
			from[i] = i - 1
		}
	}

	// Backtrack.
	var starts []int
	for i := n; i > 0; {
		starts = append(starts, from[i])
		i = from[i]
	}
	// Reverse.
	for l, r := 0, len(starts)-1; l < r; l, r = l+1, r-1 {
		starts[l], starts[r] = starts[r], starts[l]
	}
	return starts
}

func wrap(words []word, maxWidth, sentenceSpaces int, aesthetic bool) []int {
	if aesthetic {
		return wrapKnuthPlass(words, maxWidth, sentenceSpaces)
	}
	return wrapGreedy(words, maxWidth, sentenceSpaces)
}

// ---- paragraph rendering ----------------------------------------------------

// renderParagraph reflows words into lines, prepending firstPrefix on the first
// line and contPrefix on subsequent lines. Returns output lines.
func renderParagraph(words []word, firstPrefix, contPrefix string, cfg config) []string {
	if len(words) == 0 {
		// Emit a line with just the first prefix (e.g., blank prefixed line).
		line := strings.TrimRight(firstPrefix, " \t")
		return []string{line}
	}

	fpW := colWidth(firstPrefix)
	cpW := colWidth(contPrefix)

	// We must account for the prefix width when wrapping.
	// The available text width on the first line may differ from continuation
	// lines if prefix widths differ — but since our tokens are constructed so
	// that both prefixes have the same width, we use the max to be safe.
	textWidth := cfg.maxWidth - fpW
	if cpW > fpW {
		textWidth = cfg.maxWidth - cpW
	}
	if textWidth < 1 {
		textWidth = 1
	}

	starts := wrap(words, textWidth, cfg.sentenceSpaces, cfg.aesthetic)

	var lines []string
	for k, start := range starts {
		var end int
		if k+1 < len(starts) {
			end = starts[k+1]
		} else {
			end = len(words)
		}
		prefix := contPrefix
		if k == 0 {
			prefix = firstPrefix
		}
		line := prefix + joinWords(words, start, end, cfg.sentenceSpaces)
		lines = append(lines, strings.TrimRight(line, " \t"))
	}
	return lines
}

// ---- chunk processing -------------------------------------------------------

// A chunk is a contiguous run of non-blank lines (a paragraph).
// processChunk reflowes a chunk and returns output lines.
func processChunk(lines []string, cfg config) []string {
	if len(lines) == 0 {
		return nil
	}

	// Detect indentation first, then prefix within the indented text.
	indent := leadingWS(lines[0])
	afterIndent := lines[0][len(indent):]
	prefix, afterPrefix := detectPrefix(afterIndent)

	// Preserve any extra indentation between the prefix token and the text.
	afterPrefixIndent := leadingWS(afterPrefix)
	fullPrefix := indent + prefix + afterPrefixIndent

	// Detect list token from first line (after indent, prefix, and extra indent).
	listTok, _ := detectListToken(afterPrefix[len(afterPrefixIndent):])

	if listTok != "" {
		return processListChunk(lines, fullPrefix, "", cfg)
	}
	return processTextChunk(lines, fullPrefix, "", cfg)
}

// processTextChunk reflows a simple (non-list) paragraph.
func processTextChunk(lines []string, prefix, indent string, cfg config) []string {
	var out []string
	var words []word

	for _, line := range lines {
		stripped, ok := stripPrefix(line, prefix)
		if !ok {
			// Pass through: flush pending words first.
			if len(words) > 0 {
				out = append(out, renderParagraph(words, prefix+indent, prefix+indent, cfg)...)
				words = nil
			}
			out = append(out, strings.TrimRight(line, " \t"))
			continue
		}
		// A line that is blank after stripping prefix ends the paragraph
		// (shouldn't happen since we split on blank lines, but just in case).
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		// Strip per-line indentation (we always use first-line indent).
		text := strings.TrimLeft(stripped, " \t")
		words = append(words, extractWords(text, cfg)...)
	}

	if len(words) > 0 {
		out = append(out, renderParagraph(words, prefix+indent, prefix+indent, cfg)...)
	}
	return out
}

// processListChunk parses and reflows a list paragraph.
// It handles nesting: a list token that is indented more than the outer list
// starts a nested list.
func processListChunk(lines []string, prefix, baseIndent string, cfg config) []string {
	// Strip prefix from all lines, passing through any that lack it.
	type parsedLine struct {
		raw      string
		stripped string
		passThru bool
	}
	parsed := make([]parsedLine, len(lines))
	for i, l := range lines {
		stripped, ok := stripPrefix(l, prefix)
		if !ok {
			parsed[i] = parsedLine{raw: l, passThru: true}
		} else {
			parsed[i] = parsedLine{raw: l, stripped: stripped}
		}
	}

	var out []string
	i := 0
	for i < len(parsed) {
		pl := parsed[i]
		if pl.passThru {
			out = append(out, strings.TrimRight(pl.raw, " \t"))
			i++
			continue
		}
		// Blank stripped line: pass through as bare prefix.
		if strings.TrimSpace(pl.stripped) == "" {
			out = append(out, strings.TrimRight(prefix+pl.stripped, " \t"))
			i++
			continue
		}
		lineIndent := leadingWS(pl.stripped)
		afterInd := pl.stripped[len(lineIndent):]
		tok, firstText := detectListToken(afterInd)
		if tok == "" {
			// Not a list line (shouldn't happen in first iteration, may happen
			// in continuation). Treat as plain text under baseIndent.
			var words []word
			words = append(words, extractWords(afterInd, cfg)...)
			i++
			for i < len(parsed) && !parsed[i].passThru {
				next := parsed[i].stripped
				nextInd := leadingWS(next)
				nextAfter := next[len(nextInd):]
				nextTok, _ := detectListToken(nextAfter)
				if nextTok != "" {
					break
				}
				words = append(words, extractWords(strings.TrimSpace(next), cfg)...)
				i++
			}
			fp := prefix + lineIndent
			out = append(out, renderParagraph(words, fp, fp, cfg)...)
			continue
		}

		// We have a list token. Collect this item's lines.
		itemIndent := lineIndent // indentation before the token
		var itemWords []word
		itemWords = append(itemWords, extractWords(firstText, cfg)...)
		i++

		// contIndent: spaces to align continuation with text after token.
		contPad := strings.Repeat(" ", colWidth(tok))

		// Collect continuation lines (and nested lists).
		var nestedLines []string
		for i < len(parsed) {
			if parsed[i].passThru {
				break
			}
			next := parsed[i].stripped
			if strings.TrimSpace(next) == "" {
				break
			}
			nextInd := leadingWS(next)
			nextAfter := next[len(nextInd):]
			nextTok, _ := detectListToken(nextAfter)

			if nextTok != "" && len(nextInd) <= len(itemIndent) {
				// New item at same or outer level: stop.
				break
			}
			if nextTok != "" && len(nextInd) > len(itemIndent) {
				// Nested list: collect remaining lines for recursive processing.
				for i < len(parsed) && !parsed[i].passThru {
					n2 := parsed[i].stripped
					if strings.TrimSpace(n2) == "" {
						break
					}
					n2Ind := leadingWS(n2)
					n2After := n2[len(n2Ind):]
					n2Tok, _ := detectListToken(n2After)
					if n2Tok != "" && len(n2Ind) <= len(itemIndent) {
						break
					}
					// Reconstruct line with prefix removed but content kept.
					nestedLines = append(nestedLines, n2)
					i++
				}
				break
			}
			// Continuation: strip indentation to itemIndent+contPad level.
			text := strings.TrimLeft(next, " \t")
			itemWords = append(itemWords, extractWords(text, cfg)...)
			i++
		}

		// Render this item.
		firstPrefix := prefix + itemIndent + tok
		contPrefix := prefix + itemIndent + contPad
		out = append(out, renderParagraph(itemWords, firstPrefix, contPrefix, cfg)...)

		// Recursively handle nested list lines.
		if len(nestedLines) > 0 {
			// Re-add prefix for recursive call (processListChunk expects full lines).
			var nestedFull []string
			for _, nl := range nestedLines {
				nestedFull = append(nestedFull, prefix+nl)
			}
			out = append(out, processListChunk(nestedFull, prefix, baseIndent, cfg)...)
		}
	}
	return out
}

// leadingWS returns the leading whitespace of s.
func leadingWS(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}

// ---- top-level processing ---------------------------------------------------

func process(lines []string, cfg config) []string {
	var out []string

	i := 0
	for i < len(lines) {
		// Blank line: pass through.
		if strings.TrimSpace(lines[i]) == "" {
			out = append(out, "")
			i++
			continue
		}

		// Detect prefix from the first line of this paragraph.
		// The prefix token may follow indentation.
		firstIndent := leadingWS(lines[i])
		prefix, _ := detectPrefix(lines[i][len(firstIndent):])
		prefix = firstIndent + prefix // full prefix including indent

		// Collect a paragraph chunk.  Within a prefixed block a bare-prefix
		// line (containing only the token, with optional trailing whitespace)
		// acts as a paragraph separator, just like a blank line.
		j := i
		for j < len(lines) {
			line := lines[j]
			if strings.TrimSpace(line) == "" {
				break
			}
			if prefix != "" && j > i {
				stripped, ok := stripPrefix(line, prefix)
				if ok && strings.TrimSpace(stripped) == "" {
					break // bare-prefix line: end the current chunk
				}
			}
			j++
		}

		chunk := lines[i:j]
		out = append(out, processChunk(chunk, cfg)...)
		i = j

		// If we stopped at a bare-prefix line, emit it and advance.
		if i < len(lines) && prefix != "" && strings.TrimSpace(lines[i]) != "" {
			stripped, ok := stripPrefix(lines[i], prefix)
			if ok && strings.TrimSpace(stripped) == "" {
				out = append(out, strings.TrimRight(lines[i], " \t"))
				i++
			}
		}
	}
	return out
}

// ---- main -------------------------------------------------------------------

func main() {
	cfg := parseConfig()

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fp: %v\n", err)
		os.Exit(1)
	}

	// Detect newline style from first newline.
	newline := "\n"
	for i, b := range raw {
		if b == '\n' {
			if i > 0 && raw[i-1] == '\r' {
				newline = "\r\n"
			}
			break
		}
	}

	// Normalise to \n for processing.
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	// If input ends with a newline, Split produces a trailing empty string; remove it.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	out := process(lines, cfg)

	w := bufio.NewWriter(os.Stdout)
	for _, line := range out {
		fmt.Fprint(w, strings.TrimRight(line, " \t"))
		fmt.Fprint(w, newline)
	}
	w.Flush()
}
