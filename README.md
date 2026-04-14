# fp

`fp` ("fill paragraph") is a command-line text filter that reflows prose to a
given line width. It is similar to `fmt(1)` but adds support for comment
prefixes (`//`, `#`, `>`, etc.), bulleted and numbered lists, sentence spacing,
and aesthetically-balanced line breaking. See [SPEC.md](SPEC.md) for full
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

| Flag | Default | Description |
|------|---------|-------------|
| `-wN` | 80 | Maximum line width in columns |
| `-sN` | 2 | Spaces inserted after a sentence-ending character |
| `-a` | off | Aesthetic (Knuth–Plass) wrapping instead of greedy |

The same defaults can be set via environment variables:
`FP_LINE_WIDTH`, `FP_SENTENCE_SPACES`, `FP_AESTHETIC_WRAP` (non-empty to enable).

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

**Vim / Neovim** — select the paragraph in visual mode, then:

```
:'<,'>!fp
```

Or, with the cursor inside a paragraph, use the `!` motion:

```
!apfp
```

(`ap` selects "a paragraph".)

**GNU Emacs** — with a region selected:

```
C-u M-| fp RET
```

**Plan 9 text editors (sam, Acme)** - with a non-empty dot (i.e., with selected
text):

```
|fp
```

**sed / shell pipelines:**

```sh
sed -n '10,20p' file.txt | fp -w72
```
