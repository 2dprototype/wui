# wui Designer

Visual designer for wui windows. Projects are saved as WML (`.wml`) files and
can be exported as Go code. Run it with `go run ./cmd/designer [project.wml]`.

## Controls (toolbox)

| Group | Controls |
|---|---|
| Containers | Panel, GroupBox, TabControl, ScrollPanel, StatusBar |
| Text | TextEdit, EditLine, RichEdit |
| Input | ComboBox, IntUpDown, FloatUpDown, DatePicker, MonthCalendar, HotKeyEdit, IPAddressEdit |
| Buttons | Button, CheckBox, RadioButton, LinkLabel |
| Display | Label, PaintBox, ImageView, ProgressBar, Slider, ScrollBar |
| Lists | ListView, TreeView |

Every WML element type, property and event is supported, plus the `TreeView`.

## Properties

Properties are shown in collapsible groups. Besides the usual ones the
designer edits: button kind / note / default, check box three-state and
push-like, numbers-only and text alignment of edit controls, ListView view,
columns, rows, grid lines, check boxes, sorting, RichEdit word wrap / link
detection / background, MonthCalendar week numbers, ImageView image file, mode and back
color, ScrollBar vertical / range / page, window min and max size and center
on show, TabStop and a tool tip for every control.

## Events

The **Events** group lists every event of the selected control or window (the
list comes from the WML schema). Events that have code are marked with `*`.
*Edit Code...* opens the handler; the template has the exact signature the
library expects. Standard packages used in handler code (`fmt`, `time`, `os`,
`strings`, `strconv`, `math`, `sort`, `errors`, `log`) are imported
automatically in the exported Go code.

## File format

The project is a normal WML document. WML never contains code, so:

* handler code is stored in one XML comment (`wui-designer:events`),
* properties WML cannot express (tool tips, the TreeView nodes) are stored in
  a second comment (`wui-designer:props`),
* a `TreeView` is written as a `Panel` at the same place, noted in the props
  comment. An application that loads the file with package `wml` sees an
  empty Panel there, the designer brings the TreeView back.

Go export (`Ctrl+E`) writes everything, including tool tips and tree nodes.

## Not in the designer

Splitter, NativeToolBar, ToolBar, layouts (HBox / VBox / Dock / Grid), menus,
image lists and tab order have no WML representation and are set in code.