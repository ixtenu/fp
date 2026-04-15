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
fp [-wN] [-sN] [-a]
```

Reads from stdin, writes to stdout.

## Settings

`fp` has three settings which control its behavior.  The values used for these
settings come from the following sources, in priority order (highest to lowest):
command-line flags, environment variables, `.fp.ini` (project config),
`~/.config/fp/fp.ini` (user config), and finally built-in defaults.

**Command-line flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-wN` | 80 | Maximum line width in columns |
| `-sN` | 2 | Spaces inserted after a sentence-ending character |
| `-a` | off | Aesthetic (Knuth–Plass) wrapping instead of greedy |

**Environment variables:**
`FP_LINE_WIDTH`, `FP_SENTENCE_SPACES`, `FP_AESTHETIC_WRAP` (non-empty to enable).

**Project config:** a `.fp.ini` file in the current directory or any ancestor
directory (the nearest one wins).  Supports the same keys as the user config
(see below); missing keys inherit from the user config.

**User config:** `$XDG_CONFIG_HOME/fp/fp.ini` (or `~/.config/fp/fp.ini` if
`$XDG_CONFIG_HOME` is unset) with the following keys:

```ini
line_width = 80
sentence_spaces = 2
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

`fp` works with any text editor that supports filtering a region through a shell
command.

**Vim / Neovim** – select the paragraph in visual mode, then:

```
:'<,'>!fp
```

Or, with the cursor inside a paragraph, use the `!` motion:

```
!apfp
```

(`ap` selects "a paragraph".)

**GNU Emacs** – with a region selected:

```
C-u M-| fp RET
```

**Plan 9 text editors (sam, Acme)** – with a non-empty dot (i.e., with selected
text):

```
|fp
```
