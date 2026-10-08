# What was added

## Foundation (all controls)
- `Window.Invoke(f)` / `InvokeSync(f)` / `OnGUIThread()` - update the GUI from goroutines.
- Events on every control: `SetOnMouseDown/Up/Move/Wheel/DoubleClick/Enter/Leave`,
  `SetOnKeyDown/Up`, `SetOnChar`, `SetOnFocus`, `SetOnBlur`, `SetCursor`.
- `Focus()`, `HasFocus()`, `SetTabStop()` on every control; `Window.SetTabOrder(...)`.
- Layouts: `NewHBox/NewVBox` (weights), `NewDockLayout`, `NewGridLayout` (spans, weights);
  `Panel.SetLayout`, `Window.SetLayout`.
- `EnableDPIAwareness()`, `Window.DPI/ScaleFactor/Scale`, `SetOnDPIChanged`.
- Dark mode: `Window.SetDarkTitleBar`, `SetControlTheme`, `Window.ApplyDarkMode`.

## New controls
`ListView` (details/list/icons/tiles, checkboxes, multi-select, sorting, label edit,
virtual mode), `ImageList`, `NativeToolBar` (icons, toggles, drop-downs),
`ImageView`, `Splitter`, `ScrollBar`, `ScrollPanel`, `RichEdit` (formatting, RTF,
links, undo/redo), `LinkLabel`, `MonthCalendar`, `HotKeyEdit`, `IPAddressEdit`.

## Existing controls extended
- Edit/TextEdit: `SelectedText`, `ReplaceSelection`, `AppendText`, `Undo`, `Cut/Copy/Paste`,
  `LineCount`, `CursorLine`, `ScrollToCaret`, `SetCueBanner`, `SetNumbersOnly`, `SetTextAlign`.
- ComboBox: `SetEditable`, `SetOnTextChange`.
- Button: `SetKind` (split / command link), `SetNote`, `SetIcon/SetImages`, `SetDefault`,
  `SetDropDownMenu`; `Window.SetDefaultButton/SetCancelButton` (Enter / Esc).
- CheckBox: `SetThreeState`, `SetIndeterminate`, `SetPushLike`.
- ProgressBar: `SetState` (normal / error / paused).
- TreeView: `SetImages`, `SetCheckBoxes`, `SetEditable`, `SetOnExpand/Collapse/Check/LabelEdit`,
  `TreeNode.SetImage/Checked/SetChecked`.
- TabControl: `SetImages`, `AddTabWithImage`, `AddPage` (pages shown/hidden automatically).
- Menu items: `SetEnabled`, `SetRadio`, `SetImage`.

## Window and system
`SetOnMove/Activate/StateChange`, `SetFullscreen`, `SetOwner`, `Flash`, `RegisterHotKey`,
`SingleInstance(name)`, `SetTaskbarProgress`.
Dialogs: `InputDialog`, `PasswordDialog`, `TaskDialog`, `FontDialog`,
`MessageBoxYesNoCancel`, `MessageBoxRetryCancel`, `MessageBoxFor`.
Clipboard: `ClipboardFiles`, `ClipboardImage`, `SetClipboardImage`.

## wml
New element types: ListView, RichEdit, LinkLabel, MonthCalendar, HotKeyEdit,
IPAddressEdit, ImageView, ScrollPanel, ScrollBar; new properties and events on
existing ones (Kind, Note, Default, ThreeState, PushLike, NumbersOnly, TextAlign,
Editable, State, TabStop, mouse/key/focus events...).
Not available from wml (need code): NativeToolBar, Splitter, layouts, ImageList.
The designer was NOT updated.

## Not implemented
WebView2, MDI, Rebar, Pager, Expander, PropertyGrid, print/page-setup dialogs,
UI Automation/accessibility names, automatic rescaling of children on DPI change,
tooltips on NativeToolBar buttons, ListView sorting in virtual mode, TreeView
drag-reorder and multi-select, embedding the manifest as a resource, renaming
NativeToolBar buttons after creation.

See `examples/showcase`.

## Examples (all uncompiled, like the library code)
- `examples/showcase` - tool bar, ListView, RichEdit, layouts, Invoke, task dialog
- `examples/layouts` - Dock and Grid layouts with spans and weights
- `examples/scrolling` - ScrollPanel, Splitter, ScrollBar, ImageView
- `examples/tree_tabs` - TreeView icons/checkboxes/rename, TabControl pages
- `examples/buttons_dialogs` - split/command-link/default buttons, three-state check box,
  cue banner, hot key, IP, calendar, link label, input/font/yes-no-cancel dialogs
- `examples/system_features` - single instance, global hot key, fullscreen, taskbar progress,
  clipboard image/files, virtual ListView with a million rows
- `examples/wml_new_controls` - XML using the new wml elements and properties
