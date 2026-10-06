// Package wml loads window layouts from XML files at run time, similar to QML
// or XAML. A layout describes windows and controls of package wui; this
// package parses it, validates it against a schema and builds the real
// controls.
//
//	<wml version="1">
//	  <Window Name="main" Title="Login" InnerSize="400,300">
//	    <Label Text="User" Bounds="10,14,60,18"/>
//	    <EditLine Name="user" Bounds="80,10,300,22"/>
//	    <Button Name="ok" Text="Login" Bounds="300,260,90,26" OnClick="doLogin"/>
//	  </Window>
//	</wml>
//
// The work happens in three steps:
//
//  1. Parse reads the XML into a Document. It enforces size limits, rejects
//     DOCTYPEs and namespaces, and tracks line and column of every element
//     and attribute. Parse also validates, so a Document is always valid.
//     This step needs no Windows API and is safe to call from any goroutine.
//  2. Document.Build creates the wui controls, applies the properties in a
//     fixed, schema-defined order, binds event handlers by name and fills a
//     struct with the controls you want to use. Build must be called on the
//     GUI thread, like every other wui call. It is only available on Windows.
//  3. You call Show on the window.
//
// A minimal program:
//
//	type form struct {
//	    Window *wui.Window   `wml:"main"`
//	    User   *wui.EditLine `wml:"user"`
//	}
//
//	func main() {
//	    doc, err := wml.ParseFile("login.xml")
//	    if err != nil { log.Fatal(err) }
//
//	    var f form
//	    _, err = doc.Build(&f, wml.Handlers{
//	        "doLogin": func() { println(f.User.Text()) },
//	    })
//	    if err != nil { log.Fatal(err) }
//	    f.Window.Show()
//	}
//
// XML contains data only. It never contains code: events refer to Go
// functions by name, and every handler is type checked against the event it is
// bound to.
package wml
