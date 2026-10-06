package wml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

// Limits bound the size of a document so that hostile or broken input cannot
// use unbounded memory or time. A zero field means "use the default".
type Limits struct {
	MaxBytes int // size of the whole file, default 4 MiB
	MaxDepth int // element nesting depth, default 64
	MaxNodes int // number of elements, default 10000
}

// DefaultLimits are used when no WithLimits option is given.
var DefaultLimits = Limits{MaxBytes: 4 << 20, MaxDepth: 64, MaxNodes: 10000}

// maxAttrs is the most attributes one element may have.
const maxAttrs = 128

type parseConfig struct {
	file    string
	limits  Limits
	lenient bool
}

// ParseOption changes how a document is parsed.
type ParseOption func(*parseConfig)

// WithFileName sets the file name that is shown in error messages.
func WithFileName(name string) ParseOption {
	return func(c *parseConfig) { c.file = name }
}

// WithLimits replaces the default limits. Zero fields keep their default.
func WithLimits(l Limits) ParseOption {
	return func(c *parseConfig) {
		if l.MaxBytes > 0 {
			c.limits.MaxBytes = l.MaxBytes
		}
		if l.MaxDepth > 0 {
			c.limits.MaxDepth = l.MaxDepth
		}
		if l.MaxNodes > 0 {
			c.limits.MaxNodes = l.MaxNodes
		}
	}
}

// Lenient turns unknown properties into warnings (Document.Warnings) instead
// of errors. Everything else is still checked strictly.
func Lenient() ParseOption {
	return func(c *parseConfig) { c.lenient = true }
}

func newConfig(opts []ParseOption) *parseConfig {
	cfg := &parseConfig{limits: DefaultLimits}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Parse reads, checks and validates a WML document. It never panics on bad
// input. Problems are returned as an *Error or an ErrorList.
func Parse(r io.Reader, opts ...ParseOption) (*Document, error) {
	cfg := newConfig(opts)
	data, err := ioutil.ReadAll(io.LimitReader(r, int64(cfg.limits.MaxBytes)+1))
	if err != nil {
		return nil, &Error{File: cfg.file, Msg: "reading document: " + err.Error()}
	}
	if len(data) > cfg.limits.MaxBytes {
		return nil, &Error{
			File: cfg.file,
			Msg:  fmt.Sprintf("document is larger than %d bytes", cfg.limits.MaxBytes),
		}
	}
	return parseBytes(data, cfg)
}

// ParseFile parses the file at path. The path is used as the file name in
// error messages.
func ParseFile(path string, opts ...ParseOption) (*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f, append([]ParseOption{WithFileName(path)}, opts...)...)
}

// ParseFS parses the file name in fsys, for example an embed.FS. The name
// must be a valid io/fs path, so it cannot escape the file system.
func ParseFS(fsys fs.FS, name string, opts ...ParseOption) (*Document, error) {
	if !fs.ValidPath(name) {
		return nil, &Error{File: name, Msg: "invalid file path"}
	}
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f, append([]ParseOption{WithFileName(name)}, opts...)...)
}

func parseBytes(data []byte, cfg *parseConfig) (*Document, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	p := &parser{data: data, cfg: cfg, lines: lineStarts(data)}
	root, err := p.parse()
	if err != nil {
		return nil, err
	}

	c := &collector{file: cfg.file}
	if root.Type != "wml" {
		c.add(root.Line, root.Col, "the root element must be <wml>, found <%s>", root.Type)
		return nil, c.result()
	}
	for _, a := range root.Attrs {
		if a.Name != "version" {
			c.add(a.Line, a.Col, "unknown attribute %q on <wml>, only version is allowed", a.Name)
			continue
		}
		if a.Value != fmt.Sprint(CurrentVersion) {
			c.add(a.Line, a.Col, "unsupported version %q, this package understands version %d", a.Value, CurrentVersion)
		}
	}
	if root.Text != "" {
		c.add(root.Line, root.Col, "<wml> cannot contain text")
	}

	doc := &Document{File: cfg.file, Version: CurrentVersion, Windows: root.Children}
	v := &validator{c: c, lenient: cfg.lenient}
	v.document(doc)
	doc.Warnings = v.warns
	if l := c.result(); l != nil {
		return nil, l
	}
	return doc, nil
}

type frame struct {
	node *Node
	text []byte
}

type parser struct {
	data  []byte
	cfg   *parseConfig
	lines []int // byte offset of the start of every line
}

func (p *parser) lineCol(off int) (line, col int) {
	if off < 0 {
		off = 0
	}
	i := sort.Search(len(p.lines), func(i int) bool { return p.lines[i] > off }) - 1
	if i < 0 {
		i = 0
	}
	return i + 1, off - p.lines[i] + 1
}

func (p *parser) errorAt(off int, format string, args ...interface{}) *Error {
	line, col := p.lineCol(off)
	return &Error{File: p.cfg.file, Line: line, Col: col, Msg: fmt.Sprintf(format, args...)}
}

func lineStarts(data []byte) []int {
	starts := []int{0}
	for i, b := range data {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// parse turns the bytes into a tree of nodes. It stops at the first syntax
// error. Semantic checks are done later by the validator.
func (p *parser) parse() (*Node, error) {
	dec := xml.NewDecoder(bytes.NewReader(p.data))
	dec.Strict = true

	var root *Node
	var stack []*frame
	nodes := 0

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if se, ok := err.(*xml.SyntaxError); ok {
				return nil, &Error{File: p.cfg.file, Line: se.Line, Msg: "XML syntax error: " + se.Msg}
			}
			return nil, p.errorAt(int(dec.InputOffset()), "XML error: %v", err)
		}
		end := int(dec.InputOffset())

		switch t := tok.(type) {
		case xml.StartElement:
			nodes++
			if nodes > p.cfg.limits.MaxNodes {
				return nil, p.errorAt(end, "document has more than %d elements", p.cfg.limits.MaxNodes)
			}
			if len(stack)+1 > p.cfg.limits.MaxDepth {
				return nil, p.errorAt(end, "elements are nested deeper than %d levels", p.cfg.limits.MaxDepth)
			}
			if len(t.Attr) > maxAttrs {
				return nil, p.errorAt(end, "<%s> has more than %d attributes", t.Name.Local, maxAttrs)
			}
			if t.Name.Space != "" {
				return nil, p.errorAt(end, "XML namespaces are not supported")
			}

			start := bytes.LastIndexByte(p.data[:end], '<')
			if start < 0 {
				start = 0
			}
			line, col := p.lineCol(start)
			n := &Node{Type: t.Name.Local, Line: line, Col: col}

			offs := attrOffsets(p.data[start:end])
			for _, a := range t.Attr {
				if a.Name.Space != "" || a.Name.Local == "xmlns" {
					return nil, p.errorAt(start, "XML namespaces are not supported")
				}
				at := Attr{Name: a.Name.Local, Value: a.Value, Line: line, Col: col}
				if off, ok := offs[a.Name.Local]; ok {
					at.Line, at.Col = p.lineCol(start + off)
				}
				n.Attrs = append(n.Attrs, at)
			}

			if len(stack) == 0 {
				if root != nil {
					return nil, p.errorAt(start, "only one root element is allowed")
				}
				root = n
			} else {
				parent := stack[len(stack)-1].node
				parent.Children = append(parent.Children, n)
			}
			stack = append(stack, &frame{node: n})

		case xml.EndElement:
			if len(stack) == 0 {
				return nil, p.errorAt(end, "unexpected closing tag")
			}
			f := stack[len(stack)-1]
			f.node.Text = strings.TrimSpace(string(f.text))
			stack = stack[:len(stack)-1]

		case xml.CharData:
			if len(bytes.TrimSpace(t)) == 0 {
				continue
			}
			if len(stack) == 0 {
				return nil, p.errorAt(end, "text outside of the root element")
			}
			f := stack[len(stack)-1]
			if len(f.text)+len(t) > maxStringLen {
				return nil, p.errorAt(end, "text in <%s> is longer than %d bytes", f.node.Type, maxStringLen)
			}
			f.text = append(f.text, t...)

		case xml.ProcInst:
			// Only the <?xml ...?> declaration at the very start is allowed.
			if t.Target != "xml" || root != nil || len(stack) > 0 {
				return nil, p.errorAt(end, "processing instructions are not allowed")
			}

		case xml.Directive:
			return nil, p.errorAt(end, "DOCTYPE and other directives are not allowed")

		case xml.Comment:
			// Comments are ignored.
		}
	}

	if root == nil {
		return nil, &Error{File: p.cfg.file, Msg: "the document is empty, expected a <wml> root element"}
	}
	if len(stack) != 0 {
		return nil, &Error{File: p.cfg.file, Msg: "the document ends before <" + stack[len(stack)-1].node.Type + "> is closed"}
	}
	return root, nil
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// attrOffsets returns the byte offset of every attribute name inside a start
// tag, so that errors can point at the attribute and not just at the element.
// tag is the text from '<' up to and including the closing '>'.
func attrOffsets(tag []byte) map[string]int {
	offs := make(map[string]int)
	i := 1 // skip '<'
	for i < len(tag) && !isSpace(tag[i]) && tag[i] != '>' && tag[i] != '/' {
		i++ // the element name
	}
	for i < len(tag) {
		for i < len(tag) && isSpace(tag[i]) {
			i++
		}
		if i >= len(tag) || tag[i] == '>' || tag[i] == '/' {
			break
		}
		start := i
		for i < len(tag) && tag[i] != '=' && !isSpace(tag[i]) && tag[i] != '>' && tag[i] != '/' {
			i++
		}
		name := string(tag[start:i])
		if _, dup := offs[name]; !dup {
			offs[name] = start
		}
		for i < len(tag) && isSpace(tag[i]) {
			i++
		}
		if i < len(tag) && tag[i] == '=' {
			i++
			for i < len(tag) && isSpace(tag[i]) {
				i++
			}
			if i < len(tag) && (tag[i] == '"' || tag[i] == '\'') {
				quote := tag[i]
				i++
				for i < len(tag) && tag[i] != quote {
					i++
				}
				i++ // the closing quote
			}
		}
	}
	return offs
}
