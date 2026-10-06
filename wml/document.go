package wml

// CurrentVersion is the WML format version written by Marshal and the only
// version understood by this package.
const CurrentVersion = 1

// Attr is one attribute of an element.
type Attr struct {
	Name  string
	Value string
	Line  int // position of the attribute name, 0 if unknown
	Col   int
}

// Node is one element of a WML document. The element name is the control type
// ("Window", "Button", ...), attributes are properties and events, children
// are nested controls, Font and list items.
type Node struct {
	Type     string
	Attrs    []Attr
	Children []*Node
	Text     string // trimmed character data, only used by list items
	Line     int    // position of the element's '<', 0 if unknown
	Col      int
}

// Attr returns the value of the named attribute.
func (n *Node) Attr(name string) (string, bool) {
	if a := n.lookup(name); a != nil {
		return a.Value, true
	}
	return "", false
}

// SetAttr sets an attribute, replacing an existing one with the same name.
func (n *Node) SetAttr(name, value string) {
	if a := n.lookup(name); a != nil {
		a.Value = value
		return
	}
	n.Attrs = append(n.Attrs, Attr{Name: name, Value: value})
}

// Add appends a child element and returns it.
func (n *Node) Add(child *Node) *Node {
	n.Children = append(n.Children, child)
	return child
}

func (n *Node) lookup(name string) *Attr {
	for i := range n.Attrs {
		if n.Attrs[i].Name == name {
			return &n.Attrs[i]
		}
	}
	return nil
}

// Document is a parsed and validated WML file. A Document is immutable once it
// was returned by Parse: it can be shared between goroutines and built any
// number of times.
type Document struct {
	File     string   // file name used in error messages, may be empty
	Version  int      // format version, always CurrentVersion after Parse
	Windows  []*Node  // the <Window> elements in file order
	Warnings []*Error // only filled when parsing with Lenient()
}

// NewDocument creates a Document for programmatic construction. Call
// Validate before building it.
func NewDocument(windows ...*Node) *Document {
	return &Document{Version: CurrentVersion, Windows: windows}
}

// Validate checks the document against the schema. Parse already does this,
// so it is only needed for documents that were created or changed in code.
func (d *Document) Validate() error {
	c := &collector{file: d.File}
	v := &validator{c: c}
	v.document(d)
	if l := c.result(); l != nil {
		return l
	}
	return nil
}
