# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build -o fp .          # build
bash tests/run.sh         # run all tests
bash tests/run.sh <name>  # run a single test (e.g. bash tests/run.sh list-nested)
```

There is no linter configured; `go vet ./...` is sufficient for static checks.

## Architecture

The entire program is a single file, `main.go`. Processing flows in one pass:

1. **`main`** reads raw stdin, detects the newline style (LF vs CRLF), normalises to LF, then calls `process`.
2. **`process`** splits lines into blank runs (passed through) and non-blank paragraph chunks. Within a prefixed block it also recognises bare-prefix lines (e.g. `//`) as paragraph separators.
3. **`processChunk`** detects indentation, then a prefix token (via `detectPrefix`), then a list token (via `detectListToken`), and dispatches to `processTextChunk` or `processListChunk`.
4. **`processTextChunk`** collects words across all lines in the chunk, passes through any line that lacks the expected prefix, then calls `renderParagraph`.
5. **`processListChunk`** iterates over lines, collecting each list item's words. When it finds a line indented more deeply that starts with a list token, it recurses for nested lists.
6. **`renderParagraph`** calls `wrap` (greedy or Knuth–Plass), then assembles output lines with the appropriate `firstPrefix` / `contPrefix`.

### Key data structures

- **`word`** — a single whitespace-delimited token with its column width (via `go-runewidth`) and a `sentenceBreak` flag.
- **`config`** — width, sentence spaces, aesthetic flag; populated from flags and env vars.

### Sentence spacing

`wordEndsSentence` strips trailing closing quotes, checks for `.?!`, then consults the `abbreviations` map. Closing quotes recognised: `"`, `'`, `"` (U+201D), `'` (U+2019).

### Wrapping

- **Greedy** (`wrapGreedy`): standard left-to-right bin-packing.
- **Knuth–Plass** (`wrapKnuthPlass`): O(n²) DP minimising Σ(slack³) over non-final lines; the last line carries no penalty. A word that exceeds `maxWidth` alone is forced onto its own line as a fallback.

Both return a `[]int` of line-start word indices.

### Prefix detection

`detectPrefix` iterates `prefixTokens` in longest-first order (`///` before `//`, `##` before `#`). The `rem` token is matched case-insensitively. Indentation before the token is stripped first and then re-prepended to the full prefix that propagates through the rest of processing.

### Tests

Each test case in `tests/` is a `.in` file, a `.out` file (expected output), and an optional `.flags` file (command-line arguments). `tests/run.sh` runs them all. To add a test: write the `.in` and `.flags` files, run `./fp $(cat tests/<name>.flags) < tests/<name>.in > tests/<name>.out`, verify the output looks correct, then commit both files.
