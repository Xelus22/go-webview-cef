package webview

// Window hints for SetSize
const (
	HintNone  = 0
	HintFixed = 1
	HintMin   = 2
	HintMax   = 3
)

// WindowOptions contains optional settings for NewWindow
// This provides compatibility with webview library usage patterns
type WindowOptions struct {
	Title      string
	Width      int
	Height     int
	Resizable  bool
	MinWidth   int
	MinHeight  int
	MaxWidth   int
	MaxHeight  int
	Fullscreen bool
	Borderless bool
	Center     bool
}

// DialogType represents the type of dialog to show
// Used for alert/confirm/dialog compatibility
type DialogType int

const (
	DialogTypeAlert DialogType = iota
	DialogTypeConfirm
	DialogTypePrompt
	DialogTypeOpenFile
	DialogTypeSaveFile
)

// BindingResult represents the result of a JS binding call
type BindingResult struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// BindingCall represents a call from JavaScript to Go
type BindingCall struct {
	Name string        `json:"name"`
	ID   string        `json:"id"`
	Args []interface{} `json:"args"`
}

// Event represents a webview event
// Used for event handling compatibility
type Event struct {
	Type string
	Data interface{}
}

// EventType constants for common events
const (
	EventDOMReady     = "domready"
	EventNavigate     = "navigate"
	EventLoadStart    = "loadstart"
	EventLoadEnd      = "loadend"
	EventTitleChanged = "titlechanged"
	EventFocus        = "focus"
	EventBlur         = "blur"
	EventClose        = "close"
)

// SizeHint provides a type-safe way to specify size hints
// This is an alternative to the raw int constants
type SizeHint int

func (h SizeHint) Int() int {
	return int(h)
}

// Predefined size hints using the type
const (
	SizeHintNone  SizeHint = HintNone
	SizeHintFixed SizeHint = HintFixed
	SizeHintMin   SizeHint = HintMin
	SizeHintMax   SizeHint = HintMax
)
