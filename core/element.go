package core

import (
	"fmt"
	"sync"
)

// ElementType identifies the kind of UI element detected in the terminal.
type ElementType string

const (
	ElementBorder      ElementType = "border"
	ElementMenu        ElementType = "menu"
	ElementTable       ElementType = "table"
	ElementInput       ElementType = "input"
	ElementButton      ElementType = "button"
	ElementProgressBar ElementType = "progress_bar"
	ElementCheckbox    ElementType = "checkbox"
	ElementStatusBar   ElementType = "status_bar"
	ElementText        ElementType = "text"
)

// MenuItem represents an item within a menu element.
type MenuItem struct {
	RefID    string `json:"ref_id"`
	Text     string `json:"text"`
	Selected bool   `json:"selected"`
}

// NewMenuItem creates a new menu item.
func NewMenuItem(refID, text string, selected bool) MenuItem {
	return MenuItem{RefID: refID, Text: text, Selected: selected}
}

// Element represents a detected UI element within the terminal.
//
// The Rust version uses a tagged enum. In Go we use a struct with a Type
// discriminator and pointer fields for type-specific data. Only the fields
// relevant to the element's Type are non-nil.
type Element struct {
	Type  ElementType `json:"type"`
	RefID string      `json:"ref_id"`

	// Bounds is the bounding box for this element (always present).
	Bounds Bounds `json:"bounds"`

	// --- Type-specific fields (non-nil when applicable) ---

	// Border fields
	Title    *string  `json:"title,omitempty"`
	Children []string `json:"children,omitempty"`

	// Menu fields
	Items    []MenuItem `json:"items,omitempty"`
	Selected *int       `json:"selected,omitempty"`

	// Table fields
	Headers []string   `json:"headers,omitempty"`
	Rows    [][]string `json:"rows,omitempty"`

	// Input fields
	Value     *string `json:"value,omitempty"`
	CursorPos *int    `json:"cursor_pos,omitempty"`

	// Button / Checkbox / StatusBar / Text fields
	Label   *string `json:"label,omitempty"`
	Content *string `json:"content,omitempty"`
	Checked *bool   `json:"checked,omitempty"`

	// ProgressBar fields
	Percent *uint8 `json:"percent,omitempty"`
}

// String implements fmt.Stringer.
func (e Element) String() string {
	return fmt.Sprintf("Element{type:%s, ref_id:%s, bounds:%s}", e.Type, e.RefID, e.Bounds)
}

// TypeName returns the element type as a string.
func (e Element) TypeName() string {
	return string(e.Type)
}

// --- Element constructors ---

// NewBorderElement creates a bordered region element.
func NewBorderElement(refID string, bounds Bounds, title *string, children []string) Element {
	return Element{
		Type:     ElementBorder,
		RefID:    refID,
		Bounds:   bounds,
		Title:    title,
		Children: children,
	}
}

// NewMenuElement creates a menu element with selectable items.
func NewMenuElement(refID string, bounds Bounds, items []MenuItem, selected int) Element {
	return Element{
		Type:     ElementMenu,
		RefID:    refID,
		Bounds:   bounds,
		Items:    items,
		Selected: &selected,
	}
}

// NewTableElement creates a data table element.
func NewTableElement(refID string, bounds Bounds, headers []string, rows [][]string) Element {
	return Element{
		Type:    ElementTable,
		RefID:   refID,
		Bounds:  bounds,
		Headers: headers,
		Rows:    rows,
	}
}

// NewInputElement creates a text input field element.
func NewInputElement(refID string, bounds Bounds, value string, cursorPos int) Element {
	return Element{
		Type:      ElementInput,
		RefID:     refID,
		Bounds:    bounds,
		Value:     &value,
		CursorPos: &cursorPos,
	}
}

// NewButtonElement creates a clickable button element.
func NewButtonElement(refID string, bounds Bounds, label string) Element {
	return Element{
		Type:   ElementButton,
		RefID:  refID,
		Bounds: bounds,
		Label:  &label,
	}
}

// NewProgressBarElement creates a progress bar element.
func NewProgressBarElement(refID string, bounds Bounds, percent uint8) Element {
	return Element{
		Type:    ElementProgressBar,
		RefID:   refID,
		Bounds:  bounds,
		Percent: &percent,
	}
}

// NewCheckboxElement creates a checkbox/toggle element.
func NewCheckboxElement(refID string, bounds Bounds, label string, checked bool) Element {
	return Element{
		Type:    ElementCheckbox,
		RefID:   refID,
		Bounds:  bounds,
		Label:   &label,
		Checked: &checked,
	}
}

// NewStatusBarElement creates a status bar element.
func NewStatusBarElement(refID string, bounds Bounds, content string) Element {
	return Element{
		Type:    ElementStatusBar,
		RefID:   refID,
		Bounds:  bounds,
		Content: &content,
	}
}

// NewTextElement creates a generic text region element.
func NewTextElement(refID string, bounds Bounds, content string) Element {
	return Element{
		Type:    ElementText,
		RefID:   refID,
		Bounds:  bounds,
		Content: &content,
	}
}

// --- Terminal State Tree ---

// TerminalStateTree is a structured snapshot of terminal content.
type TerminalStateTree struct {
	SessionID  string     `json:"session_id"`
	Dimensions Dimensions `json:"dimensions"`
	Cursor     Position   `json:"cursor"`
	Timestamp  string     `json:"timestamp"`
	Elements   []Element  `json:"elements"`
	RawText    string     `json:"raw_text"`
	ANSIBuffer *string    `json:"ansi_buffer,omitempty"`
}

// FindElement finds an element by its reference ID, or returns nil if not found.
func (t *TerminalStateTree) FindElement(refID string) *Element {
	for i := range t.Elements {
		if t.Elements[i].RefID == refID {
			return &t.Elements[i]
		}
	}
	return nil
}

// ElementsOfType returns all elements matching the given type name.
func (t *TerminalStateTree) ElementsOfType(elementType ElementType) []*Element {
	var result []*Element
	for i := range t.Elements {
		if t.Elements[i].Type == elementType {
			result = append(result, &t.Elements[i])
		}
	}
	return result
}

// Menus returns all menu elements.
func (t *TerminalStateTree) Menus() []*Element {
	return t.ElementsOfType(ElementMenu)
}

// Tables returns all table elements.
func (t *TerminalStateTree) Tables() []*Element {
	return t.ElementsOfType(ElementTable)
}

// Inputs returns all input elements.
func (t *TerminalStateTree) Inputs() []*Element {
	return t.ElementsOfType(ElementInput)
}

// --- Ref ID Generator ---

// RefIDGenerator creates sequential reference IDs per element type.
// It produces IDs like "menu_1", "menu_2", "button_1", etc.
type RefIDGenerator struct {
	mu       sync.Mutex
	counters map[ElementType]int
}

// NewRefIDGenerator creates a new RefIDGenerator.
func NewRefIDGenerator() *RefIDGenerator {
	return &RefIDGenerator{
		counters: make(map[ElementType]int),
	}
}

// Next returns the next sequential ID for the given element type.
func (g *RefIDGenerator) Next(t ElementType) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counters[t]++
	return fmt.Sprintf("%s_%d", t, g.counters[t])
}

// Reset clears all counters, starting fresh.
func (g *RefIDGenerator) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counters = make(map[ElementType]int)
}
