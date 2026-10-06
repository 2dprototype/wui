package wml

import (
	"bytes"
	"encoding/xml"
	"strconv"
	"strings"
)

// Marshal returns the canonical XML form of the document: two space
// indentation, attributes in their original order and every special
// character escaped. Parsing the result gives an equal document, so Marshal
// is also what the designer and wmlcheck -fmt use to write files.
func (d *Document) Marshal() []byte {
	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	buf.WriteString("<wml version=\"")
	buf.WriteString(strconv.Itoa(CurrentVersion))
	buf.WriteString("\">\n")
	for _, w := range d.Windows {
		writeNode(&buf, w, 1)
	}
	buf.WriteString("</wml>\n")
	return buf.Bytes()
}

func escape(s string) string {
	var b bytes.Buffer
	// EscapeText only fails if the writer fails, and a bytes.Buffer never does.
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func writeNode(buf *bytes.Buffer, n *Node, depth int) {
	indent := strings.Repeat("  ", depth)
	buf.WriteString(indent)
	buf.WriteString("<")
	buf.WriteString(n.Type)
	for _, a := range n.Attrs {
		buf.WriteString(" ")
		buf.WriteString(a.Name)
		buf.WriteString("=\"")
		buf.WriteString(escape(a.Value))
		buf.WriteString("\"")
	}

	switch {
	case len(n.Children) == 0 && n.Text == "":
		buf.WriteString("/>\n")
	case len(n.Children) == 0:
		buf.WriteString(">")
		buf.WriteString(escape(n.Text))
		buf.WriteString("</")
		buf.WriteString(n.Type)
		buf.WriteString(">\n")
	default:
		buf.WriteString(">\n")
		for _, c := range n.Children {
			writeNode(buf, c, depth+1)
		}
		buf.WriteString(indent)
		buf.WriteString("</")
		buf.WriteString(n.Type)
		buf.WriteString(">\n")
	}
}
