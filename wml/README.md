# wml

Load window layouts from XML at run time, like QML or XAML, for the
[wui](../README.md) Windows GUI library.

```go
doc, err := wml.ParseFile("login.xml")      // 1. parse + validate (any OS, any goroutine)
var f form
view, err := doc.Build(&f, wml.Handlers{    // 2. create the controls (Windows, GUI thread)
    "doLogin": func() { /* ... */ },
})
f.Window.Show()                             // 3. show
```

## Status

| Part | State |
|---|---|
| Parser, validator, schema, `Marshal`, `wmlcheck` | done, unit tests and a fuzz test |
| Builder, handler binding, struct injection | done, Windows only |
| Designer export/import of `.xml` | not yet (the designer still saves JSON) |
| Hot reload, `<Include>`, XSD generation | not yet |

The code was written without access to a Go compiler. Run `go vet ./wml/...`
and `go test ./wml/...` (on Windows to include the builder tests) before relying
on it, and please report anything that does not compile.

## Format

```xml
<wml version="1">
  <Window Name="main" Title="Demo" InnerSize="400,300">
    <Font Name="Tahoma" Height="-11"/>
    <Label Text="Name" Position="10,12"/>
    <EditLine Name="name" Bounds="70,10,200,22" HorizontalAnchor="MinAndMax"/>
    <ComboBox Name="lang" Bounds="70,40,120,22" SelectedIndex="0">
      <Item>English</Item>
      <Item>Bangla</Item>
    </ComboBox>
    <Panel Bounds="10,80,380,100" BorderStyle="Sunken">
      <Button Text="OK" Bounds="10,10,80,26" OnClick="doOk"/>
    </Panel>
  </Window>
</wml>
```

* The root is `<wml version="1">` and contains one or more `<Window>`.
* The element name is the control type. Attributes are properties; each one
  maps to the setter of the same name (`Text` is `SetText`). `wmlcheck -schema`
  lists every element with its properties and ranges.
* `Name` is the identifier used by `View.Lookup`, `Inject` and `wml:"..."` tags.
  Letters, digits and underscores, unique in the whole file.
* `On...` attributes name a handler from `wml.Handlers`.
* Only `Window` and `Panel` contain controls. A `GroupBox` only draws a frame:
  put the controls after it in the same parent.
* `<Font Name=".." Height=".." Bold="true"/>` is allowed in any element.
* Lists (`ComboBox` items, `TabControl` tabs) are `<Item>` / `<Tab>` children.

| Kind | Syntax |
|---|---|
| bool | `true` or `false` |
| int | `25`, range checked per property |
| float | `0.5` (NaN and Inf are rejected) |
| color | `#RRGGBB` or `#RGB` |
| enum | a name from the schema, case-insensitive (`Alignment="Center"`) |
| composite | `Bounds="x,y,w,h"`, `Position="x,y"`, `Size="w,h"`, `MinMax="min,max"`, ... |

A composite cannot be combined with the properties it sets (`Bounds` and `X`
in one element is an error). Properties are applied in the order defined by the
schema, not the order in the file, so `Min`/`Max` always come before `Value`.

## Errors

All problems are collected and reported together, each with file, line and column:

```
login.xml:21:34: unknown property "Txet" on Button; did you mean "Text"?
login.xml:25:9: <GroupBox> cannot contain <Label>; a GroupBox only draws a frame, ...
```

Parse and Build return `wml.ErrorList` for several problems (`.Details()` prints
all of them) or a single `*wml.Error`.

## Safety

* XML is data. There is no code, expression or script in the format.
* Limits (defaults: 4 MiB, depth 64, 10,000 elements, 64 KiB per string;
  change with `WithLimits`).
* `DOCTYPE`, entities, processing instructions and namespaces are rejected.
* Only the types and properties in the schema can be set. Unknown names are
  errors (or warnings with `Lenient()`), never ignored silently.
* Numbers are range checked before they reach `wui`, and converted to the exact
  parameter type of the setter (an `Alpha` of 300 is rejected, not wrapped).
* Handlers are type checked against the event's setter before anything is
  created, so a wrong handler cannot produce a half built window.
* `Build` recovers panics and returns them as errors.
* Version 1 has no file or image properties, so there is no path to escape.
  `ParseFS` only accepts valid `io/fs` paths.

## API

| | |
|---|---|
| `Parse`, `ParseFile`, `ParseFS` | read and validate; options `WithFileName`, `WithLimits`, `Lenient` |
| `(*Document).Validate`, `Marshal` | check a document built in code; write canonical XML |
| `(*Document).Build(into, handlers)` | create windows and controls, fill `into` |
| `(*Document).BuildWith(opts, into, handlers)` | same, with `BuildOptions{AllowMissingHandlers}` |
| `(*View).Window`, `Lookup`, `Control`, `Inject` | access the result |
| `Lookup`, `TypeNames`, `TypeSpec`, `PropSpec` | read the schema |

`Document` is immutable after `Parse`, so you can parse once and `Build` many
times (for example one window per call).

## Tests

```
go test ./wml/...                                   # parser, validator, schema
go test -fuzz=FuzzParse -fuzztime=60s ./wml         # Go 1.18+
go test ./wml/...                                   # on Windows: also builds real controls
```

`TestSchemaMatchesWui` fails if a property or event in the schema does not
exist on the wui type, so changes in `wui` cannot silently break layouts.
