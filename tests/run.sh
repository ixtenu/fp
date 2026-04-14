#!/usr/bin/env bash
# Test harness for fp.
# Each test case consists of a .in file and a .out file in the same directory.
# An optional .flags file contains command-line arguments to pass to fp.
# Usage: ./tests/run.sh [test-name ...]  (no args = run all tests)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FP="$SCRIPT_DIR/../fp"

if [ ! -x "$FP" ]; then
    echo "fp binary not found at $FP — build it first." >&2
    exit 1
fi

pass=0
fail=0

run_test() {
    local in_file="$1"
    local base="${in_file%.in}"
    local name
    name="$(basename "$base")"
    local out_file="${base}.out"
    local flags_file="${base}.flags"

    if [ ! -f "$out_file" ]; then
        echo "SKIP  $name (no .out file)"
        return
    fi

    local flags=()
    if [ -f "$flags_file" ]; then
        read -ra flags < "$flags_file"
    fi

    local actual expected
    actual=$("$FP" "${flags[@]}" < "$in_file")
    expected=$(cat "$out_file")

    if [ "$actual" = "$expected" ]; then
        echo "PASS  $name"
        pass=$((pass + 1))
    else
        echo "FAIL  $name"
        diff <(printf '%s\n' "$expected") <(printf '%s\n' "$actual") | sed 's/^/      /'
        fail=$((fail + 1))
    fi
}

if [ "$#" -gt 0 ]; then
    # Run only the named tests.
    for name in "$@"; do
        in_file="$SCRIPT_DIR/${name%.in}.in"
        if [ ! -f "$in_file" ]; then
            echo "SKIP  $name (no .in file)" >&2
        else
            run_test "$in_file"
        fi
    done
else
    # Run all tests.
    for in_file in "$SCRIPT_DIR"/*.in; do
        run_test "$in_file"
    done
fi

echo ""
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
