// SPDX-License-Identifier: MIT

package output

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Escape returns s for a line of text output. Group, edge node, device and
// metric names and string values come from other clients of the broker, and
// a terminal acts on some characters instead of showing them. Escape
// replaces those with their Go escape sequence: the control characters (C0,
// DEL and C1, so \x1b, \n, \u009b), the bidirectional formatting characters
// (U+061C, U+200E, U+200F, U+202A to U+202E, U+2066 to U+2069), the line and
// paragraph separators (U+2028, U+2029), and each byte that is not part of
// valid UTF-8 (\xff). Every other character is kept as it is, so a plain
// name is printed unchanged. JSON output is not affected: encoding/json
// escapes what a JSON string needs.
func Escape(s string) string {
	i := firstToEscape(s)
	if i < 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	b.WriteString(s[:i])
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			b.WriteString(`\x`)
			b.WriteString(strconv.FormatUint(uint64(s[i])>>4, 16))
			b.WriteString(strconv.FormatUint(uint64(s[i])&0xf, 16))
		case escaped(r):
			q := strconv.QuoteRune(r) // '\x1b', '\n', '\u202e'
			b.WriteString(q[1 : len(q)-1])
		default:
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	return b.String()
}

// firstToEscape returns the index of the first byte of s that Escape
// replaces, or -1.
func firstToEscape(s string) int {
	for i := 0; i < len(s); {
		c := s[i]
		if c >= 0x20 && c < 0x7f {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if (r == utf8.RuneError && size == 1) || escaped(r) {
			return i
		}
		i += size
	}
	return -1
}

// escaped reports whether Escape replaces the character r.
func escaped(r rune) bool {
	switch {
	case r < 0x20, r >= 0x7f && r <= 0x9f: // C0, DEL, C1
		return true
	case r == 0x061c, r == 0x200e, r == 0x200f,
		r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069: // bidirectional formatting
		return true
	case r == 0x2028, r == 0x2029: // line and paragraph separators
		return true
	}
	return false
}
