package wui

import "syscall"

// Raw GDI procedures that are not wrapped by the w32 package.
var (
	xGdi32              = syscall.NewLazyDLL("gdi32.dll")
	xUser32             = syscall.NewLazyDLL("user32.dll")
	xMsimg32            = syscall.NewLazyDLL("msimg32.dll")
	xRoundRect          = xGdi32.NewProc("RoundRect")
	xSetPixel           = xGdi32.NewProc("SetPixel")
	xGetPixel           = xGdi32.NewProc("GetPixel")
	xSaveDC             = xGdi32.NewProc("SaveDC")
	xRestoreDC          = xGdi32.NewProc("RestoreDC")
	xCreateEllipticRgn  = xGdi32.NewProc("CreateEllipticRgn")
	xCreateRoundRectRgn = xGdi32.NewProc("CreateRoundRectRgn")
	xCreatePolygonRgn   = xGdi32.NewProc("CreatePolygonRgn")
	xGradientFill       = xMsimg32.NewProc("GradientFill")
	xDrawFocusRect      = xUser32.NewProc("DrawFocusRect")
)
