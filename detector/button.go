package detector

import (
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// buttonPattern defines open/close delimiters for a button.
type buttonPattern struct {
	open  string
	close string
}

// ButtonDetector detects clickable buttons in bracket patterns such as
// [ OK ], < Submit >, and various unicode brackets.
type ButtonDetector struct {
	patterns          []buttonPattern
	maxLabelLength    int
	shellPromptMarkers []string
}

// NewButtonDetector creates a new button detector.
func NewButtonDetector() *ButtonDetector {
	return &ButtonDetector{
		patterns: []buttonPattern{
			{open: "[ ", close: " ]"},
			{open: "[", close: "]"},
			{open: "< ", close: " >"},
			{open: "<", close: ">"},
			{open: "\u300c", close: "\u300d"}, // 「 」
		},
		maxLabelLength: 30,
		shellPromptMarkers: []string{
			"$", "#", "~", "@", ":",
			"git", "main", "master", "dev",
		},
	}
}

func (d *ButtonDetector) Name() string     { return "button" }
func (d *ButtonDetector) Priority() int    { return 60 }

func (d *ButtonDetector) Detect(grid GridReader, ctx *DetectionContext) []DetectedElement {
	var results []DetectedElement
	dims := grid.Dimensions()

	for row := 0; row < int(dims.Rows); row++ {
		buttons := d.detectButtonsInRow(grid, row)
		for _, btn := range buttons {
			if !ctx.IsRegionClaimed(btn.Bounds) {
				results = append(results, btn)
			}
		}
	}
	return results
}

func (d *ButtonDetector) detectButtonsInRow(grid GridReader, row int) []DetectedElement {
	var buttons []DetectedElement
	rowText := extractRowText(grid, row)

	if d.isShellPromptRow(rowText) {
		return buttons
	}

	for _, pat := range d.patterns {
		searchStart := 0
		for searchStart < len(rowText) {
			remaining := rowText[searchStart:]
			start := strings.Index(remaining, pat.open)
			if start < 0 {
				break
			}
			absStart := searchStart + start
			afterOpen := absStart + len(pat.open)
			if afterOpen >= len(rowText) {
				break
			}

			afterOpenStr := rowText[afterOpen:]
			end := strings.Index(afterOpenStr, pat.close)
			if end < 0 {
				break
			}

			labelStart := afterOpen
			labelEnd := afterOpen + end
			label := strings.TrimSpace(rowText[labelStart:labelEnd])

			// Validate label.
			containsDelimiters := strings.ContainsAny(label, "[]<>()\u300c\u300d")
			if label != "" && len(label) <= d.maxLabelLength && !containsDelimiters {
				buttonWidth := labelEnd + len(pat.close) - absStart
				buttonEnd := absStart + buttonWidth

				if !d.overlapsExisting(absStart, buttonEnd, buttons) {
					refID := strings.Join([]string{
						"button",
						itoa(row),
						itoa(absStart),
					}, "_")

					bounds := core.NewBounds(uint16(row), uint16(absStart), uint16(buttonWidth), 1)
					buttons = append(buttons, DetectedElement{
						Element:    core.NewButtonElement(refID, bounds, label),
						Bounds:     bounds,
						Confidence: ConfidenceHigh,
					})
				}
			}

			searchStart = absStart + len(pat.open) + end + len(pat.close)
		}
	}
	return buttons
}

func (d *ButtonDetector) isShellPromptRow(text string) bool {
	for _, marker := range d.shellPromptMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func (d *ButtonDetector) overlapsExisting(start, end int, existing []DetectedElement) bool {
	for _, btn := range existing {
		bs := int(btn.Bounds.Col)
		be := int(btn.Bounds.Col + btn.Bounds.Width)
		if !(end <= bs || start >= be) {
			return true
		}
	}
	return false
}

// itoa converts an int to its string representation without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
