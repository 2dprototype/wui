package wml

import (
	"fmt"
	"sort"
	"strings"
)

// maxErrors is the number of problems collected before validation and
// building stop reporting more.
const maxErrors = 100

// Error is a problem found in a WML document. It always carries the position
// of the offending element or attribute so that messages look like
//
//	login.xml:12:9: unknown property "Txet" on Button; did you mean "Text"?
type Error struct {
	File string // file name given with WithFileName/ParseFile, may be empty
	Line int    // 1-based, 0 if unknown
	Col  int    // 1-based byte column, 0 if unknown
	Msg  string
}

// Error implements the error interface.
func (e *Error) Error() string {
	var b strings.Builder
	if e.File != "" {
		b.WriteString(e.File)
	} else {
		b.WriteString("wml")
	}
	if e.Line > 0 {
		fmt.Fprintf(&b, ":%d", e.Line)
		if e.Col > 0 {
			fmt.Fprintf(&b, ":%d", e.Col)
		}
	}
	b.WriteString(": ")
	b.WriteString(e.Msg)
	return b.String()
}

// ErrorList is returned by Parse, Validate and Build when more than one
// problem was found. All problems are reported at once, not just the first.
type ErrorList []*Error

// Error implements the error interface. It shows the first problem and how
// many more there are; use Details to get all of them.
func (l ErrorList) Error() string {
	switch len(l) {
	case 0:
		return "no errors"
	case 1:
		return l[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", l[0].Error(), len(l)-1)
}

// Details returns every error on its own line.
func (l ErrorList) Details() string {
	lines := make([]string, len(l))
	for i, e := range l {
		lines[i] = e.Error()
	}
	return strings.Join(lines, "\n")
}

// sortByPosition orders the list by line and column.
func (l ErrorList) sortByPosition() {
	sort.SliceStable(l, func(i, j int) bool {
		if l[i].Line != l[j].Line {
			return l[i].Line < l[j].Line
		}
		return l[i].Col < l[j].Col
	})
}

// collector gathers errors up to maxErrors.
type collector struct {
	file      string
	list      []*Error
	truncated bool
}

func (c *collector) add(line, col int, format string, args ...interface{}) {
	if len(c.list) >= maxErrors {
		c.truncated = true
		return
	}
	c.list = append(c.list, &Error{
		File: c.file,
		Line: line,
		Col:  col,
		Msg:  fmt.Sprintf(format, args...),
	})
}

// result returns the collected errors as a sorted ErrorList, or nil.
func (c *collector) result() ErrorList {
	if len(c.list) == 0 {
		return nil
	}
	l := ErrorList(c.list)
	l.sortByPosition()
	if c.truncated {
		l = append(l, &Error{File: c.file, Msg: fmt.Sprintf("too many errors, only the first %d are shown", maxErrors)})
	}
	return l
}
