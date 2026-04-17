# fp

`fp` ("fill paragraph") is a command-line text filter that reflows prose to a
given line width.  It is similar to `fmt(1)` but adds support for comment
prefixes (`//`, `#`, `>`, etc.), bulleted and numbered lists, sentence spacing,
and aesthetically-balanced line breaking.  See [SPEC.md](SPEC.md) for full
behavioral details.

## Building

```sh
go build
```

## Usage

```
fp [-wN] [-sN] [-tN] [-a]
```

Reads from stdin, writes to stdout.

## Settings

`fp` has four settings which control its behavior.  The values used for these
settings come from the following sources, in priority order (highest to lowest):
command-line flags, environment variables, `.fp.ini` (project config),
`~/.config/fp/fp.ini` (user config), and finally built-in defaults.

**Command-line flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-wN` | 80 | Maximum line width in columns |
| `-sN` | 2 | Spaces inserted after a sentence-ending character |
| `-tN` | 8 | Tab stop interval in columns |
| `-a` | off | Aesthetic (Knuth–Plass) wrapping instead of greedy |

**Environment variables:**
`FP_LINE_WIDTH`, `FP_SENTENCE_SPACES`, `FP_TAB_STOP`,
`FP_AESTHETIC_WRAP` (`true`/`1`/`yes`/`on` or `false`/`0`/`no`/`off`).

**Project config:** a `.fp.ini` file in the current directory or any ancestor
directory (the nearest one wins).  Supports the same keys as the user config
(see below); missing keys inherit from the user config.

**User config:** `$XDG_CONFIG_HOME/fp/fp.ini` (or `~/.config/fp/fp.ini` if
`$XDG_CONFIG_HOME` is unset) with the following keys:

```ini
line_width = 80
sentence_spaces = 2
tab_stop = 8
aesthetic_wrap = false
```

For both the project and user config, unrecognized keys are silently ignored,
while syntax errors or invalid key values cause the entire file to be silently
ignored.

## Examples

Reflow a file to 72 columns:

```sh
fp -w72 < input.txt > output.txt
```

Reflow with single sentence spacing and aesthetic wrapping:

```sh
fp -s1 -a < input.txt
```

## Text editor integration

`fp` works with any text editor that supports filtering text through a shell
command.  It is especially useful with text editors which do not have a built-in
command for reflowing text, or which implement only a primitive version of it.

The below table shows how to filter text through `fp` with some example text
editors.  Select some text and run the indicated command (`C`=Ctrl, `M`=Alt):

| Editor | Command
| ------ | -------
| [Acme](https://github.com/9fans/plan9port) | `¦fp` |
| [dte](https://craigbarnes.gitlab.io/dte/) | `M-x filter fp` |
| [GNU Emacs](https://www.gnu.org/software/emacs/) | `C-u M-¦ fp` |
| [godit](https://github.com/nsf/godit) | `C-x ! fp` |
| [JOE](https://joe-editor.sourceforge.io/) | `C-k / fp` |
| [Kate](https://kate-editor.org/) | `C-\ fp` |
| [micro](https://micro-editor.github.io/) | `C-e textfilter fp` |
| [mle](https://github.com/adsr/mle) | `M-e fp` |
| [ne](https://github.com/vigna/ne/) | `M-t fp` |
| [NEdit](https://sourceforge.net/projects/nedit/) | `M-r fp` |
| [sam](https://github.com/japanoise/sam) | `¦fp` |
| [Textadept](https://orbitalquark.github.io/textadept/) | `C-¦ fp` |
| [Vim](https://www.vim.org/) | `:'<,'>!fp` |
| [vis](https://github.com/martanne/vis) | `:¦fp` |

Note: The above table uses `¦` (broken vertical bar) as a substitute for `|`
(vertical bar), since the latter character terminates a Markdown table cell.  In
all cases, use `|` (vertical bar) from within the text editor.
