// WML example: the layout is described in XML (login.xml) and loaded at run
// time. Run it from the repository root:
//
//	go run ./examples/wml_login                          built-in layout
//	go run ./examples/wml_login examples/wml_login/login.xml   from a file
//
// Edit login.xml and start the program again to see the change without
// recompiling. Check a file without opening a window with
//
//	go run ./cmd/wmlcheck examples/wml_login/login.xml
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/2dprototype/wui"
	"github.com/2dprototype/wui/wml"
)

// form receives the controls that the program needs. The tags are the Name
// attributes in the XML.
type form struct {
	Window   *wui.Window    `wml:"main"`
	User     *wui.EditLine  `wml:"user"`
	Pass     *wui.EditLine  `wml:"pass"`
	Remember *wui.CheckBox  `wml:"remember"`
	Lang     *wui.ComboBox  `wml:"lang"`
	Status   *wui.StatusBar `wml:"status"`
}

func main() {
	var doc *wml.Document
	var err error
	if len(os.Args) > 1 {
		doc, err = wml.ParseFile(os.Args[1])
	} else {
		doc, err = wml.Parse(strings.NewReader(layout), wml.WithFileName("login.xml"))
	}
	if err != nil {
		// Errors carry file, line and column, for example
		// login.xml:21:34: unknown property "Txet" on Button; did you mean "Text"?
		if list, ok := err.(wml.ErrorList); ok {
			log.Fatal(list.Details())
		}
		log.Fatal(err)
	}

	var f form
	_, err = doc.Build(&f, wml.Handlers{
		"doLogin": func() {
			if strings.TrimSpace(f.User.Text()) == "" {
				f.Status.SetText("Please enter a user name")
				return
			}
			f.Status.SetText(fmt.Sprintf("Welcome %s (language #%d, remember: %v)",
				f.User.Text(), f.Lang.SelectedIndex(), f.Remember.Checked()))
		},
		"doCancel": func() { f.Window.Close() },
	})
	if err != nil {
		if list, ok := err.(wml.ErrorList); ok {
			log.Fatal(list.Details())
		}
		log.Fatal(err)
	}

	if err := f.Window.Show(); err != nil {
		log.Fatal(err)
	}
}

// layout is the same text as login.xml.
const layout = `<?xml version="1.0" encoding="UTF-8"?>
<wml version="1">
  <Window Name="main" Title="WML login" InnerSize="400,300" CenterOnShow="true">
    <Font Name="Tahoma" Height="-11"/>

    <!-- A GroupBox only draws a frame. The controls that look as if they were
         inside it are its siblings, added after it. Use a Panel to really
         nest controls. -->
    <GroupBox Text="Account" Bounds="10,10,380,120" HorizontalAnchor="MinAndMax"/>
    <Label Text="User" Bounds="25,38,50,18"/>
    <EditLine Name="user" Bounds="85,35,285,22" HorizontalAnchor="MinAndMax"/>
    <Label Text="Password" Bounds="25,72,55,18"/>
    <EditLine Name="pass" Bounds="85,69,285,22" IsPassword="true" HorizontalAnchor="MinAndMax"/>
    <CheckBox Name="remember" Text="Remember me" Bounds="85,98,200,18" Checked="true"/>

    <Label Text="Language" Bounds="10,145,70,18"/>
    <ComboBox Name="lang" Bounds="85,142,150,22" SelectedIndex="0">
      <Item>English</Item>
      <Item>Bangla</Item>
    </ComboBox>

    <Button Name="cancel" Text="Cancel" Bounds="200,246,90,26"
            HorizontalAnchor="Max" VerticalAnchor="Max" OnClick="doCancel"/>
    <Button Name="ok" Text="Login" Bounds="300,246,90,26"
            HorizontalAnchor="Max" VerticalAnchor="Max" OnClick="doLogin"/>

    <StatusBar Name="status" Text="Ready" Bounds="0,278,400,22"
               HorizontalAnchor="MinAndMax" VerticalAnchor="Max"/>
  </Window>
</wml>
`
