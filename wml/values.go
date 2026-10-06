package wml

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// rgb is a parsed color value.
type rgb struct {
	R, G, B uint8
}

// parseValue parses the attribute text s for the property p. The result type
// depends on the kind:
//
//	KindString  string
//	KindBool    bool
//	KindInt     int64
//	KindFloat   float64
//	KindColor   rgb
//	KindEnum    int (index into p.Enum)
//	KindInts    []int64
//	KindFloats  []float64
func parseValue(p *PropSpec, s string) (interface{}, error) {
	if len(s) > maxStringLen {
		return nil, fmt.Errorf("value is longer than %d bytes", maxStringLen)
	}
	switch p.Kind {
	case KindString:
		return s, nil
	case KindBool:
		return parseBool(s)
	case KindInt:
		return parseInt(s, p.Min, p.Max)
	case KindFloat:
		return parseFloat(s, p.FMin, p.FMax)
	case KindColor:
		return parseColor(s)
	case KindEnum:
		return parseEnum(s, p.Enum)
	case KindInts:
		return parseInts(s, p.N, p.Min, p.Max, p.NonNegTail)
	case KindFloats:
		return parseFloats(s, p.N, p.FMin, p.FMax)
	}
	return nil, fmt.Errorf("%s is written as child elements, not as an attribute", p.Name)
}

func parseBool(s string) (bool, error) {
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%q is not a bool, use true or false", s)
}

func parseInt(s string, min, max int64) (int64, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a whole number", s)
	}
	if n < min || n > max {
		return 0, fmt.Errorf("%d is out of range, allowed are %d to %d", n, min, max)
	}
	return n, nil
}

func parseFloat(s string, min, max float64) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("%q is not a number", s)
	}
	if f < min || f > max {
		return 0, fmt.Errorf("%g is out of range, allowed are %g to %g", f, min, max)
	}
	return f, nil
}

func parseColor(s string) (rgb, error) {
	orig := s
	s = strings.TrimSpace(s)
	if len(s) == 0 || s[0] != '#' {
		return rgb{}, fmt.Errorf("%q is not a color, use #RRGGBB", orig)
	}
	s = s[1:]
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return rgb{}, fmt.Errorf("%q is not a color, use #RRGGBB", orig)
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex {
			return rgb{}, fmt.Errorf("%q is not a color, use #RRGGBB", orig)
		}
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return rgb{}, fmt.Errorf("%q is not a color, use #RRGGBB", orig)
	}
	return rgb{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}, nil
}

func parseEnum(s string, names []string) (int, error) {
	t := strings.TrimSpace(s)
	for i, name := range names {
		if strings.EqualFold(t, name) {
			return i, nil
		}
	}
	msg := fmt.Sprintf("%q is not valid, use one of: %s", s, strings.Join(names, ", "))
	if best := closest(t, names); best != "" {
		msg += fmt.Sprintf(" (did you mean %q?)", best)
	}
	return 0, fmt.Errorf("%s", msg)
}

func parseInts(s string, n int, min, max int64, nonNegTail int) ([]int64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != n {
		return nil, fmt.Errorf("%q needs %d comma separated numbers", s, n)
	}
	out := make([]int64, n)
	for i, part := range parts {
		v, err := parseInt(part, min, max)
		if err != nil {
			return nil, err
		}
		if i >= n-nonNegTail && v < 0 {
			return nil, fmt.Errorf("%d must not be negative", v)
		}
		out[i] = v
	}
	return out, nil
}

func parseFloats(s string, n int, min, max float64) ([]float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != n {
		return nil, fmt.Errorf("%q needs %d comma separated numbers", s, n)
	}
	out := make([]float64, n)
	for i, part := range parts {
		v, err := parseFloat(part, min, max)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// isIdent reports whether s is a valid control name: a letter or underscore
// followed by letters, digits and underscores, at most 64 bytes. If allowDot
// is set, dots are allowed after the first character (for handler names like
// "App.Save").
func isIdent(s string, allowDot bool) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		digit := c >= '0' && c <= '9'
		switch {
		case letter:
		case digit && i > 0:
		case allowDot && c == '.' && i > 0:
		default:
			return false
		}
	}
	return true
}

// closest returns the candidate that is most similar to name, or "" if none
// is close enough to be a plausible typo.
func closest(name string, candidates []string) string {
	maxDist := 2
	if l := len(name) / 3; l > maxDist {
		maxDist = l
	}
	best := ""
	bestDist := maxDist + 1
	lname := strings.ToLower(name)
	for _, c := range candidates {
		d := editDistance(lname, strings.ToLower(c))
		if d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = minInt(minInt(cur[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
