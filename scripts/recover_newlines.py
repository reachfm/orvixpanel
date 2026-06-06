#!/usr/bin/env python3
"""
Restore newlines in OrvixPanel Go files that were collapsed to a
single line by PowerShell's `Set-Content -NoNewline`.

Strategy: parse each file with Python, split tokens at:
  - top-level keywords: package, import, func, type, var, const
  - closing braces } that end a top-level decl
  - every ; (statement terminator)
  - every { (block opener at top level)

Then join with newlines + 2-space indent based on brace depth.
"""
import os
import re
import sys

ROOT = r"D:\orvixpanel"

# Heuristic regex: insert newline before these tokens
SPLIT_BEFORE = re.compile(
    r"(?P<sep>)(?=^|\s)(package |import |func |type |var |const |// )",
    re.MULTILINE,
)

# Better: tokenize on a small state machine.
def reformat_go(src: str) -> str:
    # If file already has newlines, leave it alone.
    if src.count("\n") > 5:
        return src
    out = []
    i = 0
    depth = 0          # brace depth
    paren_depth = 0    # paren depth (don't split inside ())
    bracket_depth = 0  # bracket depth
    line_start = True
    at_decl_start = True  # we are at the start of a new decl

    def emit(ch):
        out.append(ch)

    def newline():
        if out and out[-1] != "\n":
            out.append("\n")
        # indent
        out.append("  " * depth)

    while i < len(src):
        ch = src[i]

        # Comments — emit line as-is
        if ch == "/" and i + 1 < len(src) and src[i+1] == "/":
            # line comment
            j = src.find("\n", i)
            if j < 0:
                j = len(src)
            emit(src[i:j])
            i = j
            if i < len(src):
                newline()
            continue

        if ch == "/" and i + 1 < len(src) and src[i+1] == "*":
            # block comment
            j = src.find("*/", i)
            if j < 0:
                j = len(src)
            else:
                j += 2
            emit(src[i:j])
            i = j
            continue

        # String literal
        if ch == '"' or ch == '`':
            quote = ch
            emit(ch)
            i += 1
            while i < len(src):
                c = src[i]
                emit(c)
                if c == '\\' and i + 1 < len(src):
                    emit(src[i+1])
                    i += 2
                    continue
                if c == quote:
                    i += 1
                    break
                if c == '\n' and quote == '`':
                    # raw string literal can span lines
                    out.append("\n")
                i += 1
            continue

        # Rune literal
        if ch == "'":
            emit(ch)
            i += 1
            while i < len(src) and src[i] != "'":
                if src[i] == '\\':
                    emit(src[i:i+2])
                    i += 2
                else:
                    emit(src[i])
                    i += 1
            if i < len(src):
                emit(src[i])
                i += 1
            continue

        # Whitespace
        if ch in " \t":
            emit(ch)
            i += 1
            continue

        if ch == "\n":
            newline()
            i += 1
            continue

        # Braces
        if ch == "{":
            # If we just had a top-level decl name like "func name(" or "type X struct"
            # we want the { on the same line.
            # But if the brace is at top-level, we want it on its own line.
            # Heuristic: peek back — if non-space char is not ')', ' struct', ' interface', it's inline.
            j = len(out) - 1
            while j >= 0 and out[j] in " \t":
                j -= 1
            if j >= 0 and out[j] not in (")", ")", "\n"):
                # likely function body or struct literal at top
                # Check if we just closed a ) — if so, the { is for a function/method body
                if out[j] == ")":
                    emit(" ")
            else:
                # block at top level — put { on its own line
                if out and out[-1] != "\n":
                    out.append(" ")
                emit("{")
                depth += 1
                i += 1
                newline()
                continue
            emit("{")
            depth += 1
            i += 1
            # If the very next char is }, don't newline.
            if i < len(src) and src[i] == "}":
                emit("}")
                depth -= 1
                i += 1
            else:
                newline()
            continue

        if ch == "}":
            depth -= 1
            newline()
            emit("}")
            i += 1
            # After a top-level } (depth back to 0), newline
            if depth == 0:
                newline()
            continue

        if ch == "(":
            paren_depth += 1
            emit(ch)
            i += 1
            continue

        if ch == ")":
            paren_depth -= 1
            emit(ch)
            i += 1
            continue

        if ch == "[":
            bracket_depth += 1
            emit(ch)
            i += 1
            continue

        if ch == "]":
            bracket_depth -= 1
            emit(ch)
            i += 1
            continue

        if ch == ";":
            emit(";")
            i += 1
            # newline after ; when at depth 0 (top-level decl boundary)
            if depth == 0 and paren_depth == 0 and bracket_depth == 0:
                # but only if next non-space char isn't a continuation like `,`
                j = i
                while j < len(src) and src[j] in " \t":
                    j += 1
                if j >= len(src) or src[j] != ",":
                    newline()
            continue

        if ch == ",":
            emit(",")
            i += 1
            # After a top-level comma (e.g. var block), newline
            j = i
            while j < len(src) and src[j] in " \t":
                j += 1
            if j < len(src) and src[j] == "\n":
                pass
            elif depth == 0 and paren_depth == 0 and bracket_depth == 0:
                # top-level comma — add newline only if next token is at top level
                # actually only inside var/const blocks; otherwise inline
                pass
            continue

        # Identifiers / keywords / numbers
        # Detect top-level keywords
        if ch.isalpha() or ch == "_":
            j = i
            while j < len(src) and (src[j].isalnum() or src[j] == "_"):
                j += 1
            word = src[i:j]
            i = j

            if word in ("package", "import"):
                # If at depth 0 and we already have content, newline first
                if out and out[-1] != "\n":
                    newline()
                emit(word)
                # Consume trailing space; insert a newline after the keyword + space
                if i < len(src) and src[i] == " ":
                    emit(" ")
                    i += 1
                newline()
                continue

            if word in ("func", "type"):
                # Top-level decl — newline first if needed
                if depth == 0 and paren_depth == 0:
                    if out and out[-1] != "\n":
                        newline()
                emit(word)
                continue

            if word in ("var", "const"):
                # Top-level — newline first
                if depth == 0 and paren_depth == 0:
                    if out and out[-1] != "\n":
                        newline()
                emit(word)
                continue

            emit(word)
            continue

        # Numbers, operators, etc.
        emit(ch)
        i += 1

    return "".join(out)


def main():
    fixed = 0
    for dirpath, _dirnames, filenames in os.walk(ROOT):
        for fn in filenames:
            if not fn.endswith(".go"):
                continue
            p = os.path.join(dirpath, fn)
            with open(p, "r", encoding="utf-8", errors="replace") as f:
                src = f.read()
            if src.count("\n") > 50:
                continue  # already multi-line
            new = reformat_go(src)
            if new != src:
                with open(p, "w", encoding="utf-8", newline="\n") as f:
                    f.write(new)
                fixed += 1
                print(f"rewrote: {p}")
    print(f"fixed {fixed} files")


if __name__ == "__main__":
    main()
