// Package text shortens text without breaking it.
//
// UTF-8 encodes an umlaut in two bytes and a typographic dash or quotation mark in
// three. Cutting a German page at a byte offset therefore lands inside a character
// often enough to be seen: the reader gets a replacement character and the leftover
// byte — "…barrierefrei�n". That happened to us, on a page whose whole purpose is to
// quote someone else's text so a verdict can be checked.
package text

import "unicode/utf8"

// Truncate shortens s to at most limit runes and marks the cut. A limit below one
// returns nothing, because there is nothing sensible to show.
func Truncate(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= limit {
		return s
	}

	count := 0
	for offset := range s {
		if count == limit {
			return s[:offset] + "…"
		}
		count++
	}
	return s
}

// Snap moves a byte range outwards onto character boundaries, so a cut never lands
// inside a character. Outwards rather than inwards: an excerpt may be one character
// longer than asked for, but it is never missing the one the match started with.
func Snap(s string, start, end int) (int, int) {
	start = max(start, 0)
	end = min(end, len(s))
	if start > end {
		start = end
	}

	// Das Ende der Zeichenkette ist selbst eine gültige Grenze — dort darf nicht
	// hineingegriffen werden.
	for start > 0 && start < len(s) && !utf8.RuneStart(s[start]) {
		start--
	}
	for end < len(s) && !utf8.RuneStart(s[end]) {
		end++
	}
	return start, end
}
