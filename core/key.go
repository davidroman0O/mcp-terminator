package core

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// KeyType identifies the kind of key.
type KeyType int

const (
	KeyChar KeyType = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyEnter
	KeyTab
	KeyEscape
	KeyBackspace
	KeyDelete
	KeySpace
	KeyInsert
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyCtrl
	KeyAlt
	KeyShift
	KeyCtrlAlt
)

// Key represents a keyboard key for terminal input.
type Key struct {
	Type KeyType

	// Char is set when Type is KeyChar, KeyCtrl, KeyAlt, or KeyCtrlAlt.
	Char rune

	// Inner is set when Type is KeyShift (the modified key).
	Inner *Key
}

// String implements fmt.Stringer.
func (k Key) String() string {
	switch k.Type {
	case KeyChar:
		return string(k.Char)
	case KeyUp:
		return "Up"
	case KeyDown:
		return "Down"
	case KeyLeft:
		return "Left"
	case KeyRight:
		return "Right"
	case KeyHome:
		return "Home"
	case KeyEnd:
		return "End"
	case KeyPageUp:
		return "PageUp"
	case KeyPageDown:
		return "PageDown"
	case KeyEnter:
		return "Enter"
	case KeyTab:
		return "Tab"
	case KeyEscape:
		return "Escape"
	case KeyBackspace:
		return "Backspace"
	case KeyDelete:
		return "Delete"
	case KeySpace:
		return "Space"
	case KeyInsert:
		return "Insert"
	case KeyF1:
		return "F1"
	case KeyF2:
		return "F2"
	case KeyF3:
		return "F3"
	case KeyF4:
		return "F4"
	case KeyF5:
		return "F5"
	case KeyF6:
		return "F6"
	case KeyF7:
		return "F7"
	case KeyF8:
		return "F8"
	case KeyF9:
		return "F9"
	case KeyF10:
		return "F10"
	case KeyF11:
		return "F11"
	case KeyF12:
		return "F12"
	case KeyCtrl:
		return fmt.Sprintf("Ctrl+%c", k.Char)
	case KeyAlt:
		return fmt.Sprintf("Alt+%c", k.Char)
	case KeyShift:
		if k.Inner != nil {
			return fmt.Sprintf("Shift+%s", k.Inner)
		}
		return "Shift+?"
	case KeyCtrlAlt:
		return fmt.Sprintf("Ctrl+Alt+%c", k.Char)
	default:
		return "Unknown"
	}
}

// ParseKey parses a human-readable key string into a Key.
//
// Examples:
//   - "a" -> Key{Type: KeyChar, Char: 'a'}
//   - "Ctrl+c" -> Key{Type: KeyCtrl, Char: 'c'}
//   - "Alt+f" -> Key{Type: KeyAlt, Char: 'f'}
//   - "Enter" -> Key{Type: KeyEnter}
//   - "Up" -> Key{Type: KeyUp}
//   - "Shift+Tab" -> Key{Type: KeyShift, Inner: &Key{Type: KeyTab}}
func ParseKey(s string) (Key, error) {
	s = strings.TrimSpace(s)

	// Handle Ctrl+Alt+ modifier (must check before Ctrl+ and Alt+)
	if rest, ok := strings.CutPrefix(s, "Ctrl+Alt+"); ok {
		ch, _ := utf8.DecodeRuneInString(rest)
		if ch == utf8.RuneError || rest == "" {
			return Key{}, &InvalidInputError{Message: fmt.Sprintf("invalid Ctrl+Alt+ key: %s", s)}
		}
		return Key{Type: KeyCtrlAlt, Char: toLowerASCII(ch)}, nil
	}

	// Handle Ctrl+ modifier
	if rest, ok := strings.CutPrefix(s, "Ctrl+"); ok {
		ch, _ := utf8.DecodeRuneInString(rest)
		if ch == utf8.RuneError || rest == "" {
			return Key{}, &InvalidInputError{Message: fmt.Sprintf("invalid Ctrl+ key: %s", s)}
		}
		return Key{Type: KeyCtrl, Char: toLowerASCII(ch)}, nil
	}

	// Handle Alt+ modifier
	if rest, ok := strings.CutPrefix(s, "Alt+"); ok {
		ch, _ := utf8.DecodeRuneInString(rest)
		if ch == utf8.RuneError || rest == "" {
			return Key{}, &InvalidInputError{Message: fmt.Sprintf("invalid Alt+ key: %s", s)}
		}
		return Key{Type: KeyAlt, Char: ch}, nil
	}

	// Handle Shift+ modifier
	if rest, ok := strings.CutPrefix(s, "Shift+"); ok {
		inner, err := ParseKey(rest)
		if err != nil {
			return Key{}, err
		}
		return Key{Type: KeyShift, Inner: &inner}, nil
	}

	// Handle named keys
	switch s {
	case "Enter", "Return":
		return Key{Type: KeyEnter}, nil
	case "Tab":
		return Key{Type: KeyTab}, nil
	case "Escape", "Esc":
		return Key{Type: KeyEscape}, nil
	case "Backspace":
		return Key{Type: KeyBackspace}, nil
	case "Delete", "Del":
		return Key{Type: KeyDelete}, nil
	case "Space":
		return Key{Type: KeySpace}, nil
	case "Insert", "Ins":
		return Key{Type: KeyInsert}, nil
	case "Up":
		return Key{Type: KeyUp}, nil
	case "Down":
		return Key{Type: KeyDown}, nil
	case "Left":
		return Key{Type: KeyLeft}, nil
	case "Right":
		return Key{Type: KeyRight}, nil
	case "Home":
		return Key{Type: KeyHome}, nil
	case "End":
		return Key{Type: KeyEnd}, nil
	case "PageUp", "PgUp":
		return Key{Type: KeyPageUp}, nil
	case "PageDown", "PgDn":
		return Key{Type: KeyPageDown}, nil
	case "F1":
		return Key{Type: KeyF1}, nil
	case "F2":
		return Key{Type: KeyF2}, nil
	case "F3":
		return Key{Type: KeyF3}, nil
	case "F4":
		return Key{Type: KeyF4}, nil
	case "F5":
		return Key{Type: KeyF5}, nil
	case "F6":
		return Key{Type: KeyF6}, nil
	case "F7":
		return Key{Type: KeyF7}, nil
	case "F8":
		return Key{Type: KeyF8}, nil
	case "F9":
		return Key{Type: KeyF9}, nil
	case "F10":
		return Key{Type: KeyF10}, nil
	case "F11":
		return Key{Type: KeyF11}, nil
	case "F12":
		return Key{Type: KeyF12}, nil
	}

	// Single character
	if utf8.RuneCountInString(s) == 1 {
		ch, _ := utf8.DecodeRuneInString(s)
		return Key{Type: KeyChar, Char: ch}, nil
	}

	return Key{}, &InvalidInputError{Message: fmt.Sprintf("unknown key: %s", s)}
}

// ToEscapeSequence converts the key to terminal escape sequence bytes.
func (k Key) ToEscapeSequence() []byte {
	switch k.Type {
	case KeyChar:
		return []byte(string(k.Char))
	case KeyEnter:
		return []byte{0x0D}
	case KeyTab:
		return []byte{0x09}
	case KeyEscape:
		return []byte{0x1B}
	case KeyBackspace:
		return []byte{0x7F}
	case KeyDelete:
		return []byte{0x1B, '[', '3', '~'}
	case KeySpace:
		return []byte{0x20}
	case KeyInsert:
		return []byte{0x1B, '[', '2', '~'}
	case KeyUp:
		return []byte{0x1B, '[', 'A'}
	case KeyDown:
		return []byte{0x1B, '[', 'B'}
	case KeyRight:
		return []byte{0x1B, '[', 'C'}
	case KeyLeft:
		return []byte{0x1B, '[', 'D'}
	case KeyHome:
		return []byte{0x1B, '[', 'H'}
	case KeyEnd:
		return []byte{0x1B, '[', 'F'}
	case KeyPageUp:
		return []byte{0x1B, '[', '5', '~'}
	case KeyPageDown:
		return []byte{0x1B, '[', '6', '~'}
	case KeyF1:
		return []byte{0x1B, 'O', 'P'}
	case KeyF2:
		return []byte{0x1B, 'O', 'Q'}
	case KeyF3:
		return []byte{0x1B, 'O', 'R'}
	case KeyF4:
		return []byte{0x1B, 'O', 'S'}
	case KeyF5:
		return []byte{0x1B, '[', '1', '5', '~'}
	case KeyF6:
		return []byte{0x1B, '[', '1', '7', '~'}
	case KeyF7:
		return []byte{0x1B, '[', '1', '8', '~'}
	case KeyF8:
		return []byte{0x1B, '[', '1', '9', '~'}
	case KeyF9:
		return []byte{0x1B, '[', '2', '0', '~'}
	case KeyF10:
		return []byte{0x1B, '[', '2', '1', '~'}
	case KeyF11:
		return []byte{0x1B, '[', '2', '3', '~'}
	case KeyF12:
		return []byte{0x1B, '[', '2', '4', '~'}
	case KeyCtrl:
		// Ctrl+A = 0x01, Ctrl+Z = 0x1A
		code := byte(toLowerASCII(k.Char)) - 'a' + 1
		return []byte{code}
	case KeyAlt:
		// Alt sends ESC prefix
		seq := []byte{0x1B}
		seq = append(seq, []byte(string(k.Char))...)
		return seq
	case KeyShift:
		if k.Inner == nil {
			return nil
		}
		switch k.Inner.Type {
		case KeyTab:
			return []byte{0x1B, '[', 'Z'}
		case KeyUp:
			return []byte{0x1B, '[', '1', ';', '2', 'A'}
		case KeyDown:
			return []byte{0x1B, '[', '1', ';', '2', 'B'}
		case KeyRight:
			return []byte{0x1B, '[', '1', ';', '2', 'C'}
		case KeyLeft:
			return []byte{0x1B, '[', '1', ';', '2', 'D'}
		default:
			return k.Inner.ToEscapeSequence()
		}
	case KeyCtrlAlt:
		// Ctrl+Alt sends ESC + Ctrl code
		code := byte(toLowerASCII(k.Char)) - 'a' + 1
		return []byte{0x1B, code}
	default:
		return nil
	}
}

// toLowerASCII converts an ASCII letter to lowercase.
func toLowerASCII(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
