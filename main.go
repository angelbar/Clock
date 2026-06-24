// Clock — Reloj digital minimalista para Windows (Win32 API nativa en Go)
//
// Migrado desde la versión Python/tkinter a Go puro sin CGO.
// Ventana sin bordes, arrastrable, redimensionable, con menú hover,
// selector de colores, formato 12h/24h y persistencia de configuración.
//
// Compilación:
//   go build -ldflags="-H=windowsgui -s -w" -o Clock.exe

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// ═══════════════════════════════════════════════════════
// Win32 API bindings (vía syscall, sin CGO)
// ═══════════════════════════════════════════════════════

var (
	moduser32   = syscall.NewLazyDLL("user32.dll")
	modgdi32    = syscall.NewLazyDLL("gdi32.dll")
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")
	modcomdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	moduxtheme  = syscall.NewLazyDLL("uxtheme.dll")
)

// user32
var (
	pCreateWindowExW  = moduser32.NewProc("CreateWindowExW")
	pDefWindowProcW   = moduser32.NewProc("DefWindowProcW")
	pDestroyWindow    = moduser32.NewProc("DestroyWindow")
	pDispatchMessageW = moduser32.NewProc("DispatchMessageW")
	pGetClientRect    = moduser32.NewProc("GetClientRect")
	pGetCursorPos     = moduser32.NewProc("GetCursorPos")
	pGetDC            = moduser32.NewProc("GetDC")
	pGetMessageW      = moduser32.NewProc("GetMessageW")
	pInvalidateRect   = moduser32.NewProc("InvalidateRect")
	pKillTimer        = moduser32.NewProc("KillTimer")
	pLoadCursorW      = moduser32.NewProc("LoadCursorW")
	pMoveWindow       = moduser32.NewProc("MoveWindow")
	pPostQuitMessage  = moduser32.NewProc("PostQuitMessage")
	pRegisterClassExW = moduser32.NewProc("RegisterClassExW")
	pReleaseDC        = moduser32.NewProc("ReleaseDC")
	pScreenToClient   = moduser32.NewProc("ScreenToClient")
	pClientToScreen   = moduser32.NewProc("ClientToScreen")
	pSendMessageW     = moduser32.NewProc("SendMessageW")
	pSetTimer         = moduser32.NewProc("SetTimer")
	pSetWindowPos     = moduser32.NewProc("SetWindowPos")
	pShowWindow       = moduser32.NewProc("ShowWindow")
	pSetCursor        = moduser32.NewProc("SetCursor")
	pGetWindowRect    = moduser32.NewProc("GetWindowRect")
	pSetCapture       = moduser32.NewProc("SetCapture")
	pReleaseCapture   = moduser32.NewProc("ReleaseCapture")
	pMessageBeep      = moduser32.NewProc("MessageBeep")
)

// gdi32
var (
	pCreateFontIndirectW   = modgdi32.NewProc("CreateFontIndirectW")
	pCreateSolidBrush      = modgdi32.NewProc("CreateSolidBrush")
	pDeleteObject          = modgdi32.NewProc("DeleteObject")
	pGetStockObject        = modgdi32.NewProc("GetStockObject")
	pGetTextExtentPoint32W = modgdi32.NewProc("GetTextExtentPoint32W")
	pSelectObject          = modgdi32.NewProc("SelectObject")
	pSetBkColor            = modgdi32.NewProc("SetBkColor")
	pSetBkMode             = modgdi32.NewProc("SetBkMode")
	pSetTextColor          = modgdi32.NewProc("SetTextColor")
	pTextOutW              = modgdi32.NewProc("TextOutW")
	pMoveToEx              = modgdi32.NewProc("MoveToEx")
	pLineTo                = modgdi32.NewProc("LineTo")
	pCreatePen             = modgdi32.NewProc("CreatePen")
	pCreateCompatibleDC    = modgdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = modgdi32.NewProc("CreateCompatibleBitmap")
	pBitBlt                = modgdi32.NewProc("BitBlt")
	pDeleteDC              = modgdi32.NewProc("DeleteDC")
)

// user32 (additional functions that belong here)
var (
	pBeginPaint = moduser32.NewProc("BeginPaint")
	pEndPaint   = moduser32.NewProc("EndPaint")
	pFillRect   = moduser32.NewProc("FillRect")
)

// comdlg32
var (
	pChooseColorW = modcomdlg32.NewProc("ChooseColorW")
)

// kernel32
var (
	pGetModuleHandleW = modkernel32.NewProc("GetModuleHandleW")
	pGetLastError     = modkernel32.NewProc("GetLastError")
)

// uxtheme
var (
	pSetWindowTheme = moduxtheme.NewProc("SetWindowTheme")
)

// ═══════════════════════════════════════════════════════
// Win32 types & constants
// ═══════════════════════════════════════════════════════

type (
	HWND     uintptr
	HINST    uintptr
	HBRUSH   uintptr
	HFONT    uintptr
	HDC      uintptr
	HCURSOR  uintptr
	HMENU    uintptr
	HICON    uintptr
	HGDIOBJ  uintptr
	LPARAM   uintptr
	WPARAM   uintptr
	LRESULT  uintptr
	COLORREF uint32
)

type POINT struct {
	X, Y int32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type WNDCLASSEXW struct {
	Size        uint32
	Style       uint32
	WndProc     uintptr
	ClsExtra    int32
	WndExtra    int32
	Instance    HINST
	Icon        HICON
	Cursor      HCURSOR
	Background  HBRUSH
	MenuName    *uint16
	ClassName   *uint16
	IconSm      HICON
}

type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  WPARAM
	LParam  LPARAM
	Time    uint32
	Pt      POINT
}

type PAINTSTRUCT struct {
	Hdc         HDC
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	Reserved    [32]byte
}

type LOGFONTW struct {
	LfHeight         int32
	LfWidth          int32
	LfEscapement     int32
	LfOrientation    int32
	LfWeight         int32
	LfItalic         byte
	LfUnderline      byte
	LfStrikeOut      byte
	LfCharSet        byte
	LfOutPrecision   byte
	LfClipPrecision  byte
	LfQuality        byte
	LfPitchAndFamily byte
	LfFaceName       [32]uint16
}

type CHOOSECOLORW struct {
	LStructSize    uint32
	HwndOwner      HWND
	HInstance      HINST
	RgbResult      COLORREF
	lpCustColors   *[16]COLORREF
	Flags          uint32
	LCustData      uintptr
	LpfnHook       uintptr
	LpTemplateName *uint16
}

const (
	// Window styles
	WS_POPUP           = 0x80000000
	WS_VISIBLE         = 0x10000000
	WS_CLIPCHILDREN    = 0x02000000
	WS_CLIPSIBLINGS    = 0x04000000
	WS_SYSMENU         = 0x00080000

	// Extended styles
	WS_EX_TOPMOST      = 0x00000008
	WS_EX_LAYERED      = 0x00080000
	WS_EX_TOOLWINDOW   = 0x00000080
	WS_EX_NOACTIVATE   = 0x08000000

	// Messages
	WM_DESTROY     = 2
	WM_MOVE        = 3
	WM_SIZE        = 5
	WM_ACTIVATE    = 6
	WM_SETFOCUS    = 7
	WM_KILLFOCUS   = 8
	WM_ERASEBKGND  = 20
	WM_PAINT       = 15
	WM_CLOSE       = 16
	WM_KEYDOWN     = 256
	WM_CHAR        = 258
	WM_SYSKEYDOWN  = 260
	WM_MOUSEMOVE   = 512
	WM_LBUTTONDOWN = 513
	WM_LBUTTONUP   = 514
	WM_NCLBUTTONDOWN = 0x00A1
	WM_RBUTTONDOWN = 516
	WM_RBUTTONUP   = 517
	WM_TIMER       = 275
	WM_NCHITTEST   = 132
	WM_SETCURSOR   = 32

	// NCHITTEST results
	HTERROR       = -2
	HTTRANSPARENT = -1
	HTNOWHERE     = 0
	HTCLIENT      = 1
	HTCAPTION     = 2
	HTBOTTOM      = 15
	HTBOTTOMRIGHT = 17
	HTBOTTOMLEFT  = 16
	HTLEFT        = 10
	HTRIGHT       = 11
	HTTOP         = 12
	HTTOPLEFT     = 13
	HTTOPRIGHT    = 14

	// GDI constants
	TRANSPARENT = 1
	OPAQUE      = 2
	FW_BOLD     = 700
	ANSI_CHARSET = 0
	OUT_TT_PRECIS  = 4
	CLIP_DEFAULT_PRECIS = 0
	CLEARTYPE_QUALITY = 5
	MONO_FONT     = 48
	FF_DONTCARE   = 0
	DEFAULT_PITCH = 0
	WHITE_BRUSH   = 0
	NULL_BRUSH    = 1
	PS_SOLID      = 0
	DC_BRUSH      = 18
	BLACK_BRUSH   = 4
	SRCCOPY       = 0x00CC0020

	// Color chooser flags
	CC_RGBINIT     = 0x00000001
	CC_FULLOPEN    = 0x00000002
	CC_ANYCOLOR    = 0x00000100

	// Timer
	TIMER_CLOCK   = 1
	TIMER_HIDE    = 2
	HIDE_DELAY_MS = 300

	// Sizing
	MIN_WIDTH  = 180
	MIN_HEIGHT = 60
	BOTTOM_BAR_HEIGHT = 4
	HOVER_ZONE        = 30

	// Font sizing
	FONT_SCALE     = 100 // percent of height
	AMPM_FONT_SCALE = 75 // percent of time font

	// Window class
	CLASS_NAME = "ClockWindow"

	// Update interval
	TICK_MS = 100

	// Button constants
	BTN_W      = 50
	BTN_H      = 24
	BTN_GAP    = 2
	BTN_TOP    = 3
	BTN_RIGHT  = 4
)

var (
	NULL      = uintptr(0)
	g_hinst   HINST
	g_hwnd    HWND

	// Config
	g_cfg     Config

	// State
	g_clockFont   HFONT
	g_ampmFont    HFONT
	g_dragging    bool
	g_resizing    bool
	g_resizeEdge  int
	g_dragX       int32
	g_dragY       int32
	g_resizeW     int32
	g_resizeH     int32
	g_resizeX     int32
	g_resizeY     int32
	g_clientW     int32
	g_clientH     int32
	g_ampmMode    bool
	g_buttonsShow bool
	g_hiding      bool
	g_debug       bool
	g_hoverBtn    int   // -1 = none
	g_lastTime    string
	g_lastAmPm    string
	g_flashUntil  time.Time

	// Custom colors for color chooser
	g_custColors [16]COLORREF

	// Button definitions
	g_buttons []ButtonDef
)

// Button definitions
type ButtonDef struct {
	ID    int
	Text  string
	Rect  RECT
	Hover bool
}

const (
	BTN_CLOSE  = 0
	BTN_RESET  = 1
	BTN_PALETTE = 2
	BTN_AMPM   = 3
)

// ═══════════════════════════════════════════════════════
// Configuration
// ═══════════════════════════════════════════════════════

type Config struct {
	Bg    string `json:"bg"`
	Fg    string `json:"fg"`
	W     int32  `json:"w"`
	H     int32  `json:"h"`
	X     int32  `json:"x"` // -1 = center
	Y     int32  `json:"y"` // -1 = center
	Ampm  bool   `json:"ampm"`
}

func defaultConfig() Config {
	return Config{
		Bg:   "#222222",
		Fg:   "#F31A1A",
		W:    380,
		H:    75,
		X:    -1,
		Y:    -1,
		Ampm: false,
	}
}

func configPath() string {
	appData := os.Getenv("APPDATA")
	return filepath.Join(appData, "Clock", "config.json")
}

func loadConfig() Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		return cfg
	}
	if loaded.W < MIN_WIDTH {
		loaded.W = defaultConfig().W
	}
	if loaded.H < MIN_HEIGHT {
		loaded.H = defaultConfig().H
	}
	if loaded.Bg == "" {
		loaded.Bg = cfg.Bg
	}
	if loaded.Fg == "" {
		loaded.Fg = cfg.Fg
	}
	return loaded
}

func saveConfig() {
	var rect RECT
	pGetWindowRect.Call(uintptr(g_hwnd), uintptr(unsafe.Pointer(&rect)))
	g_cfg.X = rect.Left
	g_cfg.Y = rect.Top
	g_cfg.W = rect.Right - rect.Left
	g_cfg.H = rect.Bottom - rect.Top

	dir := filepath.Dir(configPath())
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(g_cfg, "", "  ")
	os.WriteFile(configPath(), data, 0644)
}

func resetConfig() {
	g_cfg = defaultConfig()

	// Center window
	sw := GetSystemMetrics(SM_CXSCREEN)
	sh := GetSystemMetrics(SM_CYSCREEN)
	x := (int32(sw) - g_cfg.W) / 2
	y := (int32(sh) - g_cfg.H) / 2

	pMoveWindow.Call(uintptr(g_hwnd), uintptr(uint32(x)), uintptr(uint32(y)),
		uintptr(uint32(g_cfg.W)), uintptr(uint32(g_cfg.H)), 1)

	if err := os.Remove(configPath()); err != nil {
		// Ignore if doesn't exist
	}
	g_ampmMode = false
	g_cfg.Ampm = false
	layoutButtons()

	// Force repaint
	pInvalidateRect.Call(uintptr(g_hwnd), NULL, 0)
}

func hexToColorRef(hex string) COLORREF {
	if len(hex) < 6 {
		return 0x222222
	}
	h := hex
	if h[0] == '#' {
		h = h[1:]
	}
	if len(h) != 6 {
		return 0x222222
	}
	var r, g, b byte
	for i := 0; i < 3; i++ {
		v := byte(0)
		for j := 0; j < 2; j++ {
			c := h[i*2+j]
			var n byte
			switch {
			case c >= '0' && c <= '9':
				n = c - '0'
			case c >= 'A' && c <= 'F':
				n = c - 'A' + 10
			case c >= 'a' && c <= 'f':
				n = c - 'a' + 10
			default:
				return 0x222222
			}
			v = v*16 + n
		}
		switch i {
		case 0:
			r = v
		case 1:
			g = v
		case 2:
			b = v
		}
	}
	// Win32 COLORREF is BGR
	return COLORREF(uint32(b) | uint32(g)<<8 | uint32(r)<<16)
}

func colorRefToHex(c COLORREF) string {
	r := byte((c >> 16) & 0xFF)
	g := byte((c >> 8) & 0xFF)
	b := byte(c & 0xFF)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// ═══════════════════════════════════════════════════════
// System metrics helper (via syscall since not in windows)
// ═══════════════════════════════════════════════════════

var (
	pGetSystemMetrics = moduser32.NewProc("GetSystemMetrics")
)

const (
	SM_CXSCREEN = 0
	SM_CYSCREEN = 1
)

func GetSystemMetrics(index int) int {
	ret, _, _ := pGetSystemMetrics.Call(uintptr(index))
	return int(ret)
}

// ═══════════════════════════════════════════════════════
// Color chooser
// ═══════════════════════════════════════════════════════

func pickColor(hwnd HWND, initial COLORREF) (COLORREF, bool) {
	cc := CHOOSECOLORW{
		LStructSize:  uint32(unsafe.Sizeof(CHOOSECOLORW{})),
		HwndOwner:    hwnd,
		RgbResult:    initial,
		lpCustColors: &g_custColors,
		Flags:        CC_RGBINIT | CC_FULLOPEN | CC_ANYCOLOR,
	}
	ret, _, _ := pChooseColorW.Call(uintptr(unsafe.Pointer(&cc)))
	if ret == 0 {
		return 0, false
	}
	return cc.RgbResult, true
}

// ═══════════════════════════════════════════════════════
// String helper (UTF16)
// ═══════════════════════════════════════════════════════

func utf16Ptr(s string) *uint16 {
	u := utf16.Encode([]rune(s + "\x00"))
	return &u[0]
}

func win32TextOut(hdc HDC, x, y int32, s string) {
	u := utf16.Encode([]rune(s))
	if len(u) > 0 {
		pTextOutW.Call(uintptr(hdc), uintptr(x), uintptr(y),
			uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)))
	}
}

func win32GetTextExtent(hdc HDC, s string) (int32, int32) {
	u := utf16.Encode([]rune(s))
	if len(u) == 0 {
		return 0, 0
	}
	var sz POINT
	pGetTextExtentPoint32W.Call(uintptr(hdc), uintptr(unsafe.Pointer(&u[0])),
		uintptr(len(u)), uintptr(unsafe.Pointer(&sz)))
	return sz.X, int32(sz.Y)
}

// ═══════════════════════════════════════════════════════
// Fonts
// ═══════════════════════════════════════════════════════

func createFont(height int32, bold bool) HFONT {
	lf := LOGFONTW{}
	lf.LfHeight = -height // use char height (negative = match size)
	lf.LfWeight = FW_BOLD
	if bold {
		lf.LfWeight = FW_BOLD
	} else {
		lf.LfWeight = 400
	}
	lf.LfQuality = CLEARTYPE_QUALITY
	lf.LfCharSet = ANSI_CHARSET
	lf.LfOutPrecision = OUT_TT_PRECIS
	lf.LfClipPrecision = CLIP_DEFAULT_PRECIS
	lf.LfPitchAndFamily = DEFAULT_PITCH | FF_DONTCARE

	// Face name: "Segoe UI"
	for i, c := range utf16.Encode([]rune("Segoe UI")) {
		if i < 31 {
			lf.LfFaceName[i] = c
		}
	}
	lf.LfFaceName[8] = 0

	ret, _, _ := pCreateFontIndirectW.Call(uintptr(unsafe.Pointer(&lf)))
	return HFONT(ret)
}

func calcFontSizes(clientH int32) (timeSize, ampmSize int32) {
	timeSize = clientH * FONT_SCALE / 100
	if timeSize < 12 {
		timeSize = 12
	}
	ampmSize = timeSize * AMPM_FONT_SCALE / 100
	if ampmSize < 9 {
		ampmSize = 9
	}
	return
}

func recreateFonts(hdc HDC) {
	if g_clockFont != 0 {
		pDeleteObject.Call(uintptr(g_clockFont))
	}
	if g_ampmFont != 0 {
		pDeleteObject.Call(uintptr(g_ampmFont))
	}
	timeSize, ampmSize := calcFontSizes(g_clientH)
	g_clockFont = createFont(timeSize, true)
	g_ampmFont = createFont(ampmSize, true)
}

// ═══════════════════════════════════════════════════════
// Button layout
// ═══════════════════════════════════════════════════════

func layoutButtons() {
	ampmText := "24h"
	if g_cfg.Ampm {
		ampmText = "12h"
	}
	g_buttons = []ButtonDef{
		{ID: BTN_RESET, Text: "Reset", Hover: false},
		{ID: BTN_PALETTE, Text: "Color", Hover: false},
		{ID: BTN_AMPM, Text: ampmText, Hover: false},
		{ID: BTN_CLOSE, Text: "Cerrar", Hover: false},
	}

	// Layout from right to left
	x := g_clientW - BTN_RIGHT
	for i := range g_buttons {
		idx := len(g_buttons) - 1 - i
		g_buttons[idx].Rect = RECT{
			Left:   x - BTN_W,
			Top:    BTN_TOP,
			Right:  x,
			Bottom: BTN_TOP + BTN_H,
		}
		x -= BTN_W + BTN_GAP
	}
}

// ═══════════════════════════════════════════════════════
// Drawing
// ═══════════════════════════════════════════════════════

func drawClock(hdc HDC) {
	bgColor := hexToColorRef(g_cfg.Bg)
	fgColor := hexToColorRef(g_cfg.Fg)

	// Draw background
	brush, _, _ := pCreateSolidBrush.Call(uintptr(bgColor))
	var rc RECT
	rc.Left = 0
	rc.Top = 0
	rc.Right = g_clientW
	rc.Bottom = g_clientH
	pFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&rc)), brush)
	pDeleteObject.Call(brush)

	// --- Draw clock text ---
	// Get current time
	now := time.Now()
	var timeText, ampmText string
	if g_cfg.Ampm {
		timeText = now.Format("03:04:05")
		ampmText = now.Format("PM")
	} else {
		timeText = now.Format("15:04:05")
	}

	// Select time font
	oldFont, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(g_clockFont))
	defer pSelectObject.Call(uintptr(hdc), oldFont)

	pSetTextColor.Call(uintptr(hdc), uintptr(fgColor))
	pSetBkMode.Call(uintptr(hdc), TRANSPARENT)

	// Measure time text
	tw, th := win32GetTextExtent(hdc, timeText)

	// Calculate total width for combined block (time + gap + AM/PM)
	totalW := tw
	ampmW := int32(0)
	if g_cfg.Ampm && ampmText != "" {
		prevFont, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(g_ampmFont))
		ampmW, _ = win32GetTextExtent(hdc, ampmText)
		pSelectObject.Call(uintptr(hdc), prevFont)
		totalW = tw + 5 + ampmW
	}

	// Center the combined block horizontally
	tx := (g_clientW - totalW) / 2
	if tx < 0 {
		tx = 2
	}

	// Position at top with font internal-leading correction
	// Segoe UI has ~20% internal leading above glyph; shift text up to compensate
	ty := int32(2) - th*20/100

	// Flash: hour change → blink text for 3 seconds
	drawTime := true
	if !g_flashUntil.IsZero() {
		now := time.Now()
		if now.Before(g_flashUntil) {
			drawTime = (now.UnixMilli()/250)%2 == 0
		} else {
			g_flashUntil = time.Time{} // flash expired
		}
	}

	if drawTime {
		win32TextOut(hdc, tx, ty, timeText)
	}

	// Draw AM/PM text if in 12h mode
	var ax, ay, ah int32
	if g_cfg.Ampm && ampmText != "" && drawTime {
		// Select smaller font
		oldFont2, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(g_ampmFont))
		pSetTextColor.Call(uintptr(hdc), uintptr(fgColor))
		pSetBkMode.Call(uintptr(hdc), TRANSPARENT)

		var ampAh int32
		ampmW, ampAh = win32GetTextExtent(hdc, ampmText)
		ah = ampAh

		// Position AM/PM to the right of time, aligned at bottom
		ax = tx + tw + 5 // 5px gap
		ay = ty + th - ah // align bottom

		win32TextOut(hdc, ax, ay, ampmText)

		pSelectObject.Call(uintptr(hdc), oldFont2)
	}

	// --- Draw ◢ indicator at bottom-right ---
	indicatorX := g_clientW - 14
	indicatorY := g_clientH - BOTTOM_BAR_HEIGHT - 1

	pSelectObject.Call(uintptr(hdc), oldFont)
	pSetTextColor.Call(uintptr(hdc), uintptr(fgColor))
	win32TextOut(hdc, indicatorX, indicatorY, "◢")

	// --- Draw hover buttons ---
	if g_buttonsShow {
		for _, btn := range g_buttons {
			// Fixed menu colors (ignore clock config)
			btnBg := RGB(50, 50, 50)
			btnFg := 0x00FFFFFF
			if btn.Hover {
				switch btn.ID {
				case BTN_CLOSE:
					btnBg = 0x000033CC // R:0x33 G:0x00 B:0xCC → #CC3333 (BGR)
					btnFg = 0x00FFFFFF
				default:
					btnBg = RGB(80, 80, 80)
					btnFg = 0x00FFFFFF
				}
			}

			// Draw button background
			btnBrush, _, _ := pCreateSolidBrush.Call(uintptr(btnBg))
			pFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&btn.Rect)), btnBrush)
			pDeleteObject.Call(btnBrush)

			// Draw button text
			pSetTextColor.Call(uintptr(hdc), uintptr(btnFg))
			btw, bth := win32GetTextExtent(hdc, btn.Text)
			btx := btn.Rect.Left + (btn.Rect.Right-btn.Rect.Left-btw)/2
			bty := btn.Rect.Top + (btn.Rect.Bottom-btn.Rect.Top-bth)/2
			win32TextOut(hdc, btx, bty, btn.Text)
		}
	}

	// --- Debug mode: draw 1px red borders around all elements ---
	if g_debug {
		redPen, _, _ := pCreatePen.Call(uintptr(PS_SOLID), uintptr(1), uintptr(0x000000FF))
		oldPen, _, _ := pSelectObject.Call(uintptr(hdc), redPen)
		hollowBrush, _, _ := pGetStockObject.Call(uintptr(NULL_BRUSH))
		oldBrush, _, _ := pSelectObject.Call(uintptr(hdc), hollowBrush)

		// Time text bounds
		pMoveToEx.Call(uintptr(hdc), uintptr(tx), uintptr(ty), 0)
		pLineTo.Call(uintptr(hdc), uintptr(tx+tw), uintptr(ty))
		pLineTo.Call(uintptr(hdc), uintptr(tx+tw), uintptr(ty+th))
		pLineTo.Call(uintptr(hdc), uintptr(tx), uintptr(ty+th))
		pLineTo.Call(uintptr(hdc), uintptr(tx), uintptr(ty))

		// AM/PM text bounds
		if g_cfg.Ampm && ampmText != "" {
			pMoveToEx.Call(uintptr(hdc), uintptr(ax), uintptr(ay), 0)
			pLineTo.Call(uintptr(hdc), uintptr(ax+ampmW), uintptr(ay))
			pLineTo.Call(uintptr(hdc), uintptr(ax+ampmW), uintptr(ay+ah))
			pLineTo.Call(uintptr(hdc), uintptr(ax), uintptr(ay+ah))
			pLineTo.Call(uintptr(hdc), uintptr(ax), uintptr(ay))
		}

		// Bottom bar hit zone
		bbTop := g_clientH - BOTTOM_BAR_HEIGHT
		pMoveToEx.Call(uintptr(hdc), uintptr(0), uintptr(bbTop), 0)
		pLineTo.Call(uintptr(hdc), uintptr(g_clientW), uintptr(bbTop))
		pLineTo.Call(uintptr(hdc), uintptr(g_clientW), uintptr(g_clientH))
		pLineTo.Call(uintptr(hdc), uintptr(0), uintptr(g_clientH))
		pLineTo.Call(uintptr(hdc), uintptr(0), uintptr(bbTop))

		// Hover zone (top 30px)
		pMoveToEx.Call(uintptr(hdc), uintptr(0), uintptr(HOVER_ZONE), 0)
		pLineTo.Call(uintptr(hdc), uintptr(g_clientW), uintptr(HOVER_ZONE))

		// Each button rect
		for _, btn := range g_buttons {
			pMoveToEx.Call(uintptr(hdc), uintptr(btn.Rect.Left), uintptr(btn.Rect.Top), 0)
			pLineTo.Call(uintptr(hdc), uintptr(btn.Rect.Right), uintptr(btn.Rect.Top))
			pLineTo.Call(uintptr(hdc), uintptr(btn.Rect.Right), uintptr(btn.Rect.Bottom))
			pLineTo.Call(uintptr(hdc), uintptr(btn.Rect.Left), uintptr(btn.Rect.Bottom))
			pLineTo.Call(uintptr(hdc), uintptr(btn.Rect.Left), uintptr(btn.Rect.Top))
		}

		pSelectObject.Call(uintptr(hdc), oldBrush)
		pSelectObject.Call(uintptr(hdc), oldPen)
		pDeleteObject.Call(redPen)
	}
}

func RGB(r, g, b byte) COLORREF {
	return COLORREF(uint32(b) | uint32(g)<<8 | uint32(r)<<16)
}

// ═══════════════════════════════════════════════════════
// Window procedure
// ═══════════════════════════════════════════════════════

func wndProc(hwnd HWND, msg uint32, wParam WPARAM, lParam LPARAM) LRESULT {
	switch msg {
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		// Double-buffer: draw to memory DC then BitBlt to screen
		memDC, _, _ := pCreateCompatibleDC.Call(hdc)
		bitmap, _, _ := pCreateCompatibleBitmap.Call(hdc,
			uintptr(g_clientW), uintptr(g_clientH))
		oldBmp, _, _ := pSelectObject.Call(memDC, bitmap)

		drawClock(HDC(memDC))

		pBitBlt.Call(hdc, 0, 0, uintptr(g_clientW), uintptr(g_clientH),
			memDC, 0, 0, SRCCOPY)

		pSelectObject.Call(memDC, oldBmp)
		pDeleteObject.Call(bitmap)
		pDeleteDC.Call(memDC)

		pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_ERASEBKGND:
		return 1 // We handle background entirely in WM_PAINT

	case WM_SIZE:
		g_clientW = int32(lParam & 0xFFFF)
		g_clientH = int32((lParam >> 16) & 0xFFFF)
		if g_clockFont != 0 {
			pDeleteObject.Call(uintptr(g_clockFont))
		}
		if g_ampmFont != 0 {
			pDeleteObject.Call(uintptr(g_ampmFont))
		}
		// Recreate fonts with new dimensions
		timeSize, ampmSize := calcFontSizes(g_clientH)
		g_clockFont = createFont(timeSize, true)
		g_ampmFont = createFont(ampmSize, true)
		layoutButtons()
		pInvalidateRect.Call(uintptr(hwnd), NULL, 0)
		return 0

	case WM_NCHITTEST:
		pt := POINT{
			X: int32(lParam & 0xFFFF),
			Y: int32((lParam >> 16) & 0xFFFF),
		}
		var cr RECT
		pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&cr)))
		pScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))

		// Bottom resize zone
		if pt.Y >= g_clientH-int32(BOTTOM_BAR_HEIGHT) {
			if pt.X >= g_clientW-20 {
				return HTBOTTOMRIGHT
			}
			return HTBOTTOM
		}

		// Hover buttons zone - pass through for clicks on buttons
		if g_buttonsShow {
			for _, btn := range g_buttons {
				if pt.X >= btn.Rect.Left && pt.X <= btn.Rect.Right &&
					pt.Y >= btn.Rect.Top && pt.Y <= btn.Rect.Bottom {
					return HTCLIENT
				}
			}
		}

		// Everything else: return HTCLIENT (drag handled in WM_LBUTTONDOWN)
		return HTCLIENT

	case WM_MOUSEMOVE:
		x := int32(lParam & 0xFFFF)
		y := int32((lParam >> 16) & 0xFFFF)

		// Handle dragging
		if g_dragging {
			return 0
		}

		// Handle resize
		if g_resizing {
			return 0
		}

		// Hover buttons logic
		wasShow := g_buttonsShow
		if y < HOVER_ZONE || y >= g_clientH-int32(BOTTOM_BAR_HEIGHT) {
			g_buttonsShow = true
		} else {
			// Check if mouse is over a button
			overBtn := false
			if g_buttonsShow {
				for _, btn := range g_buttons {
					if x >= btn.Rect.Left && x <= btn.Rect.Right &&
						y >= btn.Rect.Top && y <= btn.Rect.Bottom {
						overBtn = true
						break
					}
				}
			}
			if !overBtn {
				if !g_hiding {
					g_hiding = true
					pSetTimer.Call(uintptr(hwnd), TIMER_HIDE, HIDE_DELAY_MS, NULL)
				}
			}
		}

		// Update button hover state
		if g_buttonsShow {
			for i := range g_buttons {
				was := g_buttons[i].Hover
				g_buttons[i].Hover = x >= g_buttons[i].Rect.Left &&
					x <= g_buttons[i].Rect.Right &&
					y >= g_buttons[i].Rect.Top &&
					y <= g_buttons[i].Rect.Bottom
				if g_buttons[i].Hover != was {
					pInvalidateRect.Call(uintptr(hwnd), NULL, 0)
				}
			}
		}

		if g_buttonsShow != wasShow {
			if g_buttonsShow {
				// Cancel hide timer
				if g_hiding {
					pKillTimer.Call(uintptr(hwnd), TIMER_HIDE)
					g_hiding = false
				}
			}
			pInvalidateRect.Call(uintptr(hwnd), NULL, 0)
		}
		return 0

	case WM_LBUTTONDOWN:
		x := int32(lParam & 0xFFFF)
		y := int32((lParam >> 16) & 0xFFFF)

		// Check button clicks
		if g_buttonsShow {
			for _, btn := range g_buttons {
				if x >= btn.Rect.Left && x <= btn.Rect.Right &&
					y >= btn.Rect.Top && y <= btn.Rect.Bottom {
					switch btn.ID {
					case BTN_CLOSE:
						closeWin()
					case BTN_RESET:
						resetConfig()
					case BTN_PALETTE:
						pickColors()
					case BTN_AMPM:
						toggleAmpm()
					}
					return 0
				}
			}
		}

		// Start drag for main area (not on resize bar, not on buttons)
		if y < g_clientH-int32(BOTTOM_BAR_HEIGHT) &&
			(y >= HOVER_ZONE || !g_buttonsShow) {
			pReleaseCapture.Call()
			pSendMessageW.Call(uintptr(hwnd), WM_NCLBUTTONDOWN, HTCAPTION, 0)
		}
		return 0

	case WM_KEYDOWN:
		if wParam == VK_ESCAPE {
			closeWin()
			return 0
		}
		return 0

	case WM_TIMER:
		switch wParam {
		case TIMER_CLOCK:
			// Update clock display
			now := time.Now()
			var timeText string
			if g_cfg.Ampm {
				timeText = now.Format("03:04:05")
			} else {
				timeText = now.Format("15:04:05")
			}

			// Detect hour change → start 3-second flash
			if g_lastTime != "" && timeText != g_lastTime &&
				len(timeText) >= 2 && (len(g_lastTime) < 2 || timeText[:2] != g_lastTime[:2]) {
				g_flashUntil = now.Add(3 * time.Second)
			}

			// Repaint: on time change OR during flash
			if timeText != g_lastTime || (!g_flashUntil.IsZero() && now.Before(g_flashUntil)) {
				if timeText != g_lastTime {
					g_lastTime = timeText
				}
				pInvalidateRect.Call(uintptr(hwnd), NULL, 0)
			}
		case TIMER_HIDE:
			g_hiding = false
			g_buttonsShow = false
			g_hoverBtn = -1
			pInvalidateRect.Call(uintptr(hwnd), NULL, 0)
		}
		return 0

	case WM_ACTIVATE:
		// Window can be covered by others (no TOPMOST)
		return 0

	case WM_CLOSE:
		closeWin()
		return 0

	case WM_DESTROY:
		pKillTimer.Call(uintptr(hwnd), TIMER_CLOCK)
		pKillTimer.Call(uintptr(hwnd), TIMER_HIDE)
		if g_clockFont != 0 {
			pDeleteObject.Call(uintptr(g_clockFont))
		}
		if g_ampmFont != 0 {
			pDeleteObject.Call(uintptr(g_ampmFont))
		}
		pPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := pDefWindowProcW.Call(uintptr(hwnd), uintptr(msg),
		uintptr(wParam), uintptr(lParam))
	return LRESULT(ret)
}

// ═══════════════════════════════════════════════════════
// Key constants
// ═══════════════════════════════════════════════════════

const (
	VK_ESCAPE       = 0x1B
	SWP_NOMOVE      = 0x0002
	SWP_NOSIZE      = 0x0001
	SWP_NOACTIVATE  = 0x0010
	HWND_TOPMOST    = HWND(^uintptr(1)) // -1
	HWND_NOTOPMOST  = HWND(^uintptr(2)) // -2
	SW_SHOWNORMAL   = 1
	SW_SHOWNOACTIVATE = 4
	WM_CREATE       = 1
)

// ═══════════════════════════════════════════════════════
// Actions
// ═══════════════════════════════════════════════════════

func closeWin() {
	saveConfig()
	pDestroyWindow.Call(uintptr(g_hwnd))
}

func pickColors() {
	// Pick background color
	initialBg := hexToColorRef(g_cfg.Bg)
	if col, ok := pickColor(g_hwnd, initialBg); ok {
		g_cfg.Bg = colorRefToHex(col)
	}
	// Pick foreground color
	initialFg := hexToColorRef(g_cfg.Fg)
	if col, ok := pickColor(g_hwnd, initialFg); ok {
		g_cfg.Fg = colorRefToHex(col)
	}
	saveConfig()
	pInvalidateRect.Call(uintptr(g_hwnd), NULL, 0)
}

func toggleAmpm() {
	g_cfg.Ampm = !g_cfg.Ampm
	layoutButtons()
	pInvalidateRect.Call(uintptr(g_hwnd), NULL, 0)
}

// ═══════════════════════════════════════════════════════
// Main
// ═══════════════════════════════════════════════════════

func init() {
	runtime.LockOSThread()
}

func main() {
	// Parse command-line flags
	for _, arg := range os.Args[1:] {
		if arg == "-debug" {
			g_debug = true
		}
	}

	g_cfg = loadConfig()
	g_ampmMode = g_cfg.Ampm

	// Get instance handle
	inst, _, _ := pGetModuleHandleW.Call(NULL)
	g_hinst = HINST(inst)

	// Register window class
	className := utf16Ptr(CLASS_NAME)
	wc := WNDCLASSEXW{
		Size:       uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:      0,
		WndProc:    syscall.NewCallback(wndProc),
		Instance:   g_hinst,
		Cursor:     loadCursor(HINST(0), IDC_ARROW),
		Background: 0, // We handle background in WM_PAINT
		ClassName:  className,
	}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// Calculate position
	sw := GetSystemMetrics(SM_CXSCREEN)
	sh := GetSystemMetrics(SM_CYSCREEN)
	x := g_cfg.X
	y := g_cfg.Y
	if x < 0 || y < 0 {
		x = (int32(sw) - g_cfg.W) / 2
		y = (int32(sh) - g_cfg.H) / 2
	}

	// Create window
	style := uint32(WS_POPUP | WS_VISIBLE | WS_CLIPCHILDREN | WS_CLIPSIBLINGS)
	exStyle := uint32(WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE)

	hwnd, _, _ := pCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("Clock"))),
		uintptr(style),
		uintptr(uint32(x)), uintptr(uint32(y)),
		uintptr(uint32(g_cfg.W)), uintptr(uint32(g_cfg.H)),
		NULL, NULL, uintptr(g_hinst), NULL)

	if hwnd == 0 {
		fmt.Fprintf(os.Stderr, "CreateWindowEx failed: %d\n", getLastError())
		os.Exit(1)
	}

	g_hwnd = HWND(hwnd)

	// Disable visual styles for this window (helps with custom painting)
	pSetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(" "))),
		uintptr(unsafe.Pointer(utf16Ptr(" "))))

	// Layout buttons
	g_clientW = g_cfg.W
	g_clientH = g_cfg.H
	layoutButtons()

	// Create fonts
	timeSize, ampmSize := calcFontSizes(g_clientH)
	g_clockFont = createFont(timeSize, true)
	g_ampmFont = createFont(ampmSize, true)

	// Start clock timer (100ms)
	pSetTimer.Call(hwnd, TIMER_CLOCK, TICK_MS, NULL)

	// Show window
	pShowWindow.Call(hwnd, SW_SHOWNOACTIVATE)

	// Message loop
	var msg MSG
	for {
		ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// ═══════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════

func getLastError() uint32 {
	ret, _, _ := pGetLastError.Call()
	return uint32(ret)
}

const IDC_ARROW = 32512

func loadCursor(inst HINST, id int) HCURSOR {
	ret, _, _ := pLoadCursorW.Call(uintptr(inst), uintptr(id))
	return HCURSOR(ret)
}
