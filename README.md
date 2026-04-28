# Windows GUI Library (wui)

A pure Go library to create native Windows GUIs. This project is a modified and enhanced version of the original [gonutz/wui](https://github.com/gonutz/wui) library.

See [the online documentation](https://pkg.go.dev/github.com/2dprototype/wui) for API details.

## Credits

This project is based on the excellent work by [gonutz](https://github.com/gonutz) who created the original [wui](https://github.com/gonutz/wui) library. We've extended it with additional features and improvements while maintaining compatibility with the original API.

Original library: [github.com/gonutz/wui](https://github.com/gonutz/wui)

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