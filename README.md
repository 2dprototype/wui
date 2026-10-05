# Windows GUI Library (wui)

A pure Go library to create native Windows GUIs. This project is a modified and enhanced version of the original [gonutz/wui](https://github.com/gonutz/wui) library.

See [the online documentation](https://pkg.go.dev/github.com/2dprototype/wui) for API details.

## Credits

This project is based on the excellent work by [gonutz](https://github.com/gonutz) who created the original [wui](https://github.com/gonutz/wui) library. We've extended it with additional features and improvements while maintaining compatibility with the original API.

Original library: [github.com/gonutz/wui](https://github.com/gonutz/wui)

## Self-contained

The former third-party packages are now part of this module, so there are no
external dependencies and no network access is needed to build:

- `github.com/2dprototype/wui/w32` (Win32 bindings, from gonutz/w32 v2, extended)
- `github.com/2dprototype/wui/check` (test helper, from gonutz/check)

## New Controls

- `GroupBox` - captioned frame for visually grouping controls
- `TabControl` - tab headers with `SetOnChange`; show/hide `Panel`s per tab (use `ContentBounds()` to place them)
- `StatusBar` - bottom status bar, optional multiple parts via `SetParts` / `SetPartText`
- `Window.AddTimer(ms, f)` - periodic `Timer` running on the GUI thread (`Start`, `Stop`, `SetInterval`)
- `DatePicker` - date / long date / time chooser with `Value`, `SetValue`, `SetOnChange`
- `ColorDialog` - system color chooser (`SetColor`, `Execute(parent)`, `Color`)
- `ClipboardText()` / `SetClipboardText(text)` - read and write unicode text on the clipboard

## Centering Windows

- `window.SetCenterOnShow(true)` - centers on the screen when shown with `Show`, and over the parent window when shown with `ShowModal`
- `window.Center()`, `window.CenterOnScreen()`, `window.CenterOnParent()` - center at any time

## Examples

The `examples` folder has one small program for each new feature: `group_box`, `tab_control`, `status_bar`, `timer`, `center_window`, `clipboard`, `color_dialog`, `date_picker`, `canvas_drawing`, `canvas_mouse`, `tree_view`, `context_menu`, `tooltip` and `toolbar`. Run one with `go run ./examples/timer`.

## Canvas Drawing (PaintBox)

New `Canvas` functions, all work inside `PaintBox.SetOnPaint`:

- **Strokes**: `SetStroke(wui.Stroke{Width, Style, Cap, Join})`, `SetLineWidth`, `SetLineStyle` - width, dash styles (`LineDash`, `LineDot`, ...), round/square/flat caps, round/bevel/miter joins. They apply to every outline function (`Line`, `DrawRect`, `DrawEllipse`, `Polyline`, `Arc`, `DrawPie`, ...)
- **Shapes**: `DrawRoundRect`, `FillRoundRect`, `FillRoundRectOutline`, `FillRectOutline`, `FillEllipseOutline`, `DrawPolygon`, `FillPolygonOutline`, `Bezier`, `Curve` (smooth curve through points), `SetPixel`, `Pixel`, `DrawFocusRect`
- **Helpers**: `RegularPolygonPoints` (triangle, hexagon, ...), `StarPoints`, `Clear`
- **Fills**: `FillGradientH`, `FillGradientV`, `FillRectAlpha` (transparent fill)
- **Images**: `DrawImageScaled` (stretch), `DrawImageAlpha` (transparency)
- **Clipping**: `PushDrawEllipse`, `PushDrawRoundRect`, `PushDrawPolygon` (undo with `PopDrawRegion`)
- **State**: `Save()` / `Restore()` for stroke, font and clip region
- **Text**: `TextRectEllipsis` (cuts long text with "...")
- **Mouse**: `PaintBox.SetOnMouseDown`, `SetOnMouseUp`, `SetOnDoubleClick` (with `MouseButton`); dragging keeps reporting `SetOnMouseMove` outside the box

## More Controls and Features

- `TreeView` / `TreeNode` - hierarchical list (`Add`, `Selected`, `SetOnSelect`, `SetOnDoubleClick`, `Expand`, `Collapse`, `ExpandAll`, `Remove`, `Tag`)
- `PopupMenu` - context menus; `Window.SetContextMenu(menu)` and `control.SetContextMenu(menu)` open on right click, or call `menu.Show(window, x, y)`
- `SetToolTip(text)` - hover hint, available on every control
- `ToolBar` - `NewToolBar(parent, x, y, height)` with `AddButton` and `AddSeparator`

New examples: `canvas_drawing`, `canvas_mouse`, `tree_view`, `context_menu`, `tooltip`, `toolbar`.

## Minimal Example

This is all the code you need to create a window (which does not do much):

```Go
package main

import "github.com/2dprototype/wui"

func main() {
	wui.NewWindow().Show()
}
```

## The Designer

A graphical designer tool is included for visually creating GUIs. It is located at `github.com/2dprototype/wui/cmd/designer`.

### Features

- **Visual Layout**: Drag and drop controls onto a window
- **Property Editor**: Modify control properties in real-time
- **Live Preview**: See your GUI as you build it
- **Code Generation**: Export designs as Go source code
- **Project Management**: Save and load designs as JSON project files
- **Font Customization**: Configure fonts for individual controls
- **Event Handling**: Define event handlers (OnPaint, etc.)
- **Multiple Controls**: Button, Label, CheckBox, RadioButton, Slider, ProgressBar, EditLine, TextEdit, ComboBox, Panel, PaintBox, IntUpDown, FloatUpDown

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+N` | New Project |
| `Ctrl+O` | Open Project (JSON) |
| `Ctrl+S` | Save Project (JSON) |
| `Ctrl+Shift+S` | Save Project As (JSON) |
| `Ctrl+E` | Export as Go Code |
| `Ctrl+J` | Export as JSON |
| `F5` | Run Preview |
| `Del` | Delete Selected Control |
| `F1` | About |
| `Ctrl+F1` | Keyboard Shortcuts Reference |

### Project File Format

Projects are saved as JSON files with the following structure:

```json
{
  "windows": [
    {
      "name": "window",
      "type": "Window",
      "props": [
        {"name": "Title", "value": "My Window"},
        {"name": "InnerWidth", "value": 400},
        {"name": "InnerHeight", "value": 300}
      ],
      "font": {
        "name": "Tahoma",
        "height": -11,
        "bold": false,
        "italic": false,
        "underlined": false,
        "striked_out": false
      },
      "children": [
        {
          "name": "button1",
          "type": "Button",
          "props": [
            {"name": "Text", "value": "Click Me"},
            {"name": "X", "value": 10},
            {"name": "Y", "value": 10},
            {"name": "Width", "value": 100},
            {"name": "Height", "value": 25}
          ]
        }
      ]
    }
  ],
  "events": [
    {
      "control_name": "paintBox1",
      "event_name": "OnPaint",
      "code": "func(canvas *wui.Canvas) {\n\tcanvas.FillRect(0, 0, 100, 100, wui.RGB(255, 0, 0))\n}"
    }
  ]
}
```

## Installation

```bash
go get github.com/2dprototype/wui
```

## Requirements

- Go 1.16 or later
- Windows operating system (uses native Windows API)

## Changes from Original

This version includes several enhancements over the original [gonutz/wui](https://github.com/gonutz/wui):

- **JSON Project Files**: Save and load complete GUI designs

## License

This project maintains the same license as the original work. See the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for bugs and feature requests.

## Acknowledgments

Special thanks to [gonutz](https://github.com/gonutz) for creating the original wui library that made this project possible.