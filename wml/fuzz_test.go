//go:build go1.18
// +build go1.18

package wml

import (
	"bytes"
	"testing"
)

// FuzzParse checks the two promises that matter for untrusted input: Parse
// never panics, and everything it accepts can be written out and parsed again
// to the exact same result.
func FuzzParse(f *testing.F) {
	f.Add([]byte(loginDoc))
	f.Add([]byte(`<wml><Window/></wml>`))
	f.Add([]byte(`<wml version="1"><Window Name="w"><Button Text="x" Bounds="1,2,3,4" OnClick="h"/></Window></wml>`))
	f.Add([]byte(`<wml><Window><ComboBox><Item>a</Item><Item/></ComboBox></Window></wml>`))
	f.Add([]byte(`<!DOCTYPE x [<!ENTITY a "b">]><wml/>`))
	f.Add([]byte(`<wml xmlns="x"><Window/></wml>`))
	f.Add([]byte("\xef\xbb\xbf<wml><Window/></wml>"))

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := Parse(bytes.NewReader(data), WithLimits(Limits{MaxBytes: 1 << 16}))
		if err != nil {
			return
		}
		out := doc.Marshal()
		doc2, err := Parse(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("marshaled output does not parse: %v\n%s", err, out)
		}
		if out2 := doc2.Marshal(); !bytes.Equal(out, out2) {
			t.Fatalf("Marshal is not stable:\n%s\n---\n%s", out, out2)
		}
	})
}
