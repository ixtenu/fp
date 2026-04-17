# fp specification

`fp` ("fill paragraph") is a command-line text filter program similar to
`fmt(1)` but implementing additional features, e.g., prefix and list handling.

## Usage

```
fp [-wN] [-sN] [-tN] [-a]
Reads from stdin and writes to standard out.
-wN Set maximum line width to N columns (default 80).
-sN Set spaces after sentence end (default 2).
-tN Set tab stop interval to N columns (default 8).
-a Wrap lines aesthetically rather than greedily.
```

The defaults for options can also be set in configuration files or via
environment variables.  The priority order, from lowest to highest, is:
built-in defaults, user configuration file, project configuration file,
environment variables, command-line flags.

### User configuration file

`fp` reads `$XDG_CONFIG_HOME/fp/fp.ini` if `XDG_CONFIG_HOME` is set, otherwise
`~/.config/fp/fp.ini`.  The file uses INI syntax: `key = value` pairs, with `#`
or `;` introducing comments.  Unrecognized keys are ignored.  If the file
contains invalid syntax or an invalid value for a recognised key, the entire
file is ignored.

Supported keys:

| Key | Type | Description |
|-----|------|-------------|
| `line_width` | integer | Maximum line width in columns |
| `sentence_spaces` | integer | Spaces inserted after a sentence-ending character |
| `tab_stop` | integer | Tab stop interval in columns |
| `aesthetic_wrap` | boolean (`true`/`false`, `1`/`0`, `yes`/`no`, `on`/`off`) | Enable aesthetic (Knuth–Plass) wrapping |

Example:

```ini
# fp configuration
line_width = 100
sentence_spaces = 1
tab_stop = 8
aesthetic_wrap = true
```

### Project configuration file

`fp` searches for a `.fp.ini` file starting from the current working directory
and walking upward toward the filesystem root, stopping at the first file found.
This file uses the same INI syntax and supports the same keys as the user
configuration file.  Keys not present in `.fp.ini` retain the values from the
user configuration file (or built-in defaults if no user configuration file
exists).  An invalid `.fp.ini` is silently ignored.

### Environment variables

- `FP_LINE_WIDTH`
- `FP_SENTENCE_SPACES`
- `FP_TAB_STOP`
- `FP_AESTHETIC_WRAP` (non-empty to enable)

## Paragraphs

Blank lines separate paragraphs.  If multiple blank lines exist between
paragraphs, they are passed through to the output unchanged.  Each paragraph is
reflowed independently.

## Trailing whitespace

All output lines are stripped of trailing whitespace, even if the corresponding
input lines had trailing whitespace.

## Sentence spacing

When spaces between sentences is >1, detecting where sentences end becomes
important.  The following characters end a sentence: `".?!"`.  Sentence-end
characters are only regarded as ending a sentence when they are immediately
followed by a space, newline, or quotation mark.  If a sentence-end character
occurs immediately before a quotation mark, the spacing is added after the
quotation mark.  Quotation marks for this purpose include ASCII `"` and `'` and
the Unicode closing curly quotes `"` and `'`.

As an exception, several English-language abbreviations which almost never occur
at the end of a sentence are detected and the sentence-end characters in them
are ignored; these include:

`Mr.`, `Ms.`, `Mrs.`, `Dr.`, `Prof.`, `St.`, `vs.`, `approx.`, `fig.`,
`vol.`, `e.g.`, `i.e.`, `viz.`

## "Aesthetic" wrap mode

By default, `fp` wraps each line greedily, putting as many characters on each
line as possible before reaching the maximum line width.  At times, this results
in lines wrapped in ugly ways.  In aesthetic wrap mode, lines are wrapped using
the Knuth–Plass algorithm (without hyphenation) to minimise the overall badness
of the paragraph's line breaks, similar to what TeX does for paragraph
formatting.

## File encoding

UTF-8 input is supported.  Right-to-left text is unsupported.

Double-width Unicode characters (e.g., CJK) are counted as two columns for line
wrapping purposes.

The original newlines (Unix LF or DOS CRLF) are preserved.  If the input has
mixed newlines, the outputted newlines will match the first newline in the
input.

## Indentation

If the input text is indented, this is preserved in the output.  The exact
whitespace characters used on the first line of the paragraph are used for all
lines of the reflowed paragraph.  Tabs, spaces, or a mixture of both are
supported.  Tabs are measured using the configured tab stop interval (default
8 columns).

## Prefixing

If the first input line starts with any of the following tokens followed by a
space, it is treated as a prefix which must occur at the start of each output
line:

`>`, `//`, `///`, `--`, `#`, `##`, `%`, `!`, `*`, `rem` (case-insensitive)

When multiple tokens could match (e.g., `///` vs. `//`), the longest match is
used.

The `>` token is for Markdown block quotations.  The other tokens are intended
to implement support for wrapping source code comments for a wide variety of
languages.  Since the token must occur on the first input line, this works
reasonably well even though `fp` is unaware of which programming language is
being used.

If the prefix token is indented, that indentation is preserved in the output.

If there is extra indentation between the prefix token and the text (e.g.,
`//  text` with two spaces after `//`), that extra indentation is also preserved:
it becomes part of the effective prefix, so all output lines in that paragraph
carry the same extra indentation.

If an input line does not begin with the expected prefix, it is passed through
to the output as-is (without reflowing).  This handles the case where a user
accidentally includes non-comment lines in the selection.

### Paragraph breaks within prefixed blocks

Within a prefixed block, a paragraph break occurs when a line contains only the
prefix token (with optional trailing whitespace) and no following text, or when
a completely blank (unprefixed) line is encountered.

## Lists

If any of the following tokens followed by a space occur at the start of the
first input line (possibly following a prefix token and any indentation), then
the input is understood to be a list:

- `-`
- `+`
- `#.` (used in some extended Markdown dialects to denote an auto-numbered list)
- `[0-9]+\.` (one or more digits followed by a period)

When the input is regarded as a list, if the same token or pattern recurs at
the start of another line (at the same indentation level), it is treated as a
new list item.  For numbered lists (`[0-9]+\.`), any number triggers a new item;
the numbers need not be sequential.  The `#.` token is preserved literally in
the output and is never converted to an actual number.

Text in a list item is wrapped so that continuation lines align with the first
character following the token and its trailing space.  For example:

```
- List item text that wraps
  to the next line here.

42. List item text that wraps
    to the next line here.
```

### Nested lists

Lists may be nested.  A nested list is one whose list token is indented by more
than the enclosing list's token.  Continuation lines of each list item align
with the first character of that item's text, as above.

### Lists inside prefixed blocks

Lists may occur inside prefixed lines.  Each output line begins with the prefix
(and any prefix indentation), followed by the list content.  For example:

```
// - First item text that may
//   wrap to the next line.
// - Second item.
```

If the list token is itself indented within the prefix, that indentation is
also preserved.
