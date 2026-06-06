#!/usr/bin/env python3
"""Brute-force reformat: insert newlines after each `;` and before each
top-level decl keyword. Run on all .go files that have fewer than
20 lines (i.e. ones PowerShell mangled)."""
import os, re

ROOT = r"D:\orvixpanel"

def reformat(src: str) -> str:
    # If the file has 50+ lines, it's healthy.
    if src.count("\n") > 50:
        return src
    # Insert \n after every `;` that isn't already followed by \n.
    src = re.sub(r";\s*(?!\n)", ";\n", src)
    # Insert \n before every top-level decl keyword.
    # Match: start of string OR \n OR \s followed by one of these.
    src = re.sub(r"(^|[\s])(package|func|type|var|const)\s", r"\1\n\2 ", src, flags=re.MULTILINE)
    # Insert \n after every `{` at depth 0 — but brace tracking is
    # expensive; instead, insert \n before every `}` so closing braces
    # at top level land on their own line.
    src = re.sub(r"\s*\}", "\n}", src)
    # Insert \n before every `{` that's at start of a block (after = or
    # in struct/func signature). Heuristic: look for `){` (end of
    # arg list, start of body) → keep { on same line; everything else
    # at top level gets newline.
    # Skip — too risky.
    # Collapse multiple newlines.
    src = re.sub(r"\n{3,}", "\n\n", src)
    # Ensure file ends with a newline.
    if not src.endswith("\n"):
        src += "\n"
    return src


def main():
    n = 0
    for dirpath, _, filenames in os.walk(ROOT):
        for fn in filenames:
            if not fn.endswith(".go"):
                continue
            p = os.path.join(dirpath, fn)
            with open(p, "r", encoding="utf-8", errors="replace") as f:
                src = f.read()
            new = reformat(src)
            if new != src:
                with open(p, "w", encoding="utf-8", newline="\n") as f:
                    f.write(new)
                n += 1
    print(f"reformatted {n} files")


if __name__ == "__main__":
    main()
