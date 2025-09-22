package hotkey

import (
	"fmt"
	"strconv"
	"strings"
)

var modifierAliases = map[string]uintptr{
	"ALT":     MOD_ALT,
	"OPTION":  MOD_ALT,
	"CTRL":    MOD_CONTROL,
	"CONTROL": MOD_CONTROL,
	"SHIFT":   MOD_SHIFT,
	"WIN":     MOD_WIN,
	"WINDOWS": MOD_WIN,
	"SUPER":   MOD_WIN,
}

// ParseCombination converts a human readable combination (e.g. "Ctrl+Alt+T")
// into the modifier and virtual-key codes expected by the Win32 RegisterHotKey API.
func ParseCombination(combo string) (uintptr, uintptr, error) {
	trimmed := strings.TrimSpace(combo)
	if trimmed == "" {
		return 0, 0, fmt.Errorf("hotkey combination is empty")
	}

	parts := strings.Split(trimmed, "+")
	keyToken := strings.TrimSpace(parts[len(parts)-1])
	if keyToken == "" {
		return 0, 0, fmt.Errorf("hotkey key is missing")
	}

	var modifiers uintptr
	for _, part := range parts[:len(parts)-1] {
		token := strings.ToUpper(strings.TrimSpace(part))
		if token == "" {
			continue
		}
		alias, ok := modifierAliases[token]
		if !ok {
			return 0, 0, fmt.Errorf("unsupported modifier %q", part)
		}
		modifiers |= alias
	}

	key, err := parseKeyToken(keyToken)
	if err != nil {
		return 0, 0, err
	}

	return modifiers, key, nil
}

// FormatCombination renders modifiers and key codes back to a canonical string representation.
func FormatCombination(modifiers, key uintptr) string {
	var parts []string
	if modifiers&MOD_CONTROL != 0 {
		parts = append(parts, "Ctrl")
	}
	if modifiers&MOD_ALT != 0 {
		parts = append(parts, "Alt")
	}
	if modifiers&MOD_SHIFT != 0 {
		parts = append(parts, "Shift")
	}
	if modifiers&MOD_WIN != 0 {
		parts = append(parts, "Win")
	}
	parts = append(parts, describeKeyToken(key))
	return strings.Join(parts, "+")
}

// NormalizeCombination validates and canonicalises a combination string.
func NormalizeCombination(combo string) (string, error) {
	modifiers, key, err := ParseCombination(combo)
	if err != nil {
		return "", err
	}
	return FormatCombination(modifiers, key), nil
}

func parseKeyToken(token string) (uintptr, error) {
	normalized := strings.ToUpper(strings.TrimSpace(token))
	if normalized == "" {
		return 0, fmt.Errorf("hotkey key is empty")
	}

	if len(normalized) == 1 {
		ch := normalized[0]
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			return uintptr(ch), nil
		}
	}

	switch normalized {
	case "SPACE":
		return 0x20, nil
	case "TAB":
		return 0x09, nil
	case "ENTER", "RETURN":
		return 0x0D, nil
	case "ESC", "ESCAPE":
		return 0x1B, nil
	}

	if strings.HasPrefix(normalized, "F") {
		if number, err := strconv.Atoi(normalized[1:]); err == nil {
			if number >= 1 && number <= 24 {
				return uintptr(0x70 + number - 1), nil
			}
		}
	}

	return 0, fmt.Errorf("unsupported hotkey key %q", token)
}

func describeKeyToken(key uintptr) string {
	if key >= 'A' && key <= 'Z' {
		return string(rune(key))
	}
	if key >= '0' && key <= '9' {
		return string(rune(key))
	}
	if key >= 0x70 && key <= 0x87 {
		return fmt.Sprintf("F%d", key-0x70+1)
	}

	switch key {
	case 0x20:
		return "Space"
	case 0x09:
		return "Tab"
	case 0x0D:
		return "Enter"
	case 0x1B:
		return "Esc"
	}

	return fmt.Sprintf("VK_%X", key)
}
