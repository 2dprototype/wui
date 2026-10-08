// Loads ui.xml, which uses the controls and properties added to wml. Run it
// from the repository root:
//
//	go run ./examples/wml_new_controls
//
// Check the file without opening a window:
//
//	go run ./cmd/wmlcheck examples/wml_new_controls/ui.xml
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/2dprototype/wui"
	"github.com/2dprototype/wui/wml"
)

type form struct {
	Window *wui.Window   `wml:"main"`
	List   *wui.ListView `wml:"list"`
	Rich   *wui.RichEdit `wml:"rich"`
}

func main() {
	doc, err := wml.ParseFile("examples/wml_new_controls/ui.xml")
	if err != nil {
		log.Fatal(err)
	}
	var f form
	_, err = doc.Build(&f, wml.Handlers{
		"onSave":     func() { fmt.Println("save") },
		"onFocus":    func() { fmt.Println("focus") },
		"onText":     func() { fmt.Println("text changed") },
		"onSelect":   func(i int) { fmt.Println("selected", i) },
		"onActivate": func(i int) { fmt.Println("activated", i) },
		"onLink":     func(url string) { wui.OpenURL(url) },
		"onDate":     func(t time.Time) { fmt.Println(t.Format("2006-01-02")) },
	})
	if err != nil {
		log.Fatal(err)
	}
	f.Window.Show()
}
