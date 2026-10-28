// Package routing decides which employee or queue should handle a request,
// using the organization's approved rules, topics, job profiles and live
// availability, and explains every decision.
package routing

import (
	"strings"
	"unicode"
)

// persian maps Arabic code points commonly typed or transcribed in
// Persian text to their Persian forms, so keywords match either way.
var persian = strings.NewReplacer("ي", "ی", "ك", "ک", "ة", "ه", "ۀ", "ه", "أ", "ا", "إ", "ا", "‌", " ",
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9")

// Normalize lowercases text, unifies Persian/Arabic letters and digits and
// collapses everything that is not a letter or digit into single spaces.
func Normalize(s string) string {
	s = persian.Replace(strings.ToLower(s))
	f := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.Is(unicode.Mn, r) })
	return " " + strings.Join(f, " ") + " "
}

// Contains reports whether normalized text contains the keyword as whole
// word(s) or, for keywords of 4+ letters, as a word prefix (so "refund"
// matches "refunds" and "بازپرداخت" matches "بازپرداختش").
func Contains(normText, keyword string) bool {
	k := strings.TrimSpace(Normalize(keyword))
	if k == "" {
		return false
	}
	if strings.Contains(normText, " "+k+" ") {
		return true
	}
	return len([]rune(k)) >= 4 && strings.Contains(normText, " "+k)
}

// matches returns the keywords found in normalized text.
func matches(normText string, keywords []string) []string {
	var out []string
	for _, k := range keywords {
		if Contains(normText, k) {
			out = append(out, k)
		}
	}
	return out
}
