package render

import (
	"strings"
	"testing"
)

func TestHTMLRendersScrollableGanttWithoutBarLabels(t *testing.T) {
	canMeetDeadline := false
	payload := map[string]any{
		"plan": map[string]any{
			"items": []any{
				map[string]any{
					"teamName":        "A组",
					"styleNo":         "M260727010046",
					"customerName":    "客户<甲>&",
					"allocatedQty":    120,
					"efficiency":      0.85,
					"startDate":       "2026-08-03",
					"endDate":         "2026-08-03",
					"canMeetDeadline": canMeetDeadline,
				},
			},
		},
	}

	got := HTML(payload)
	for _, want := range []string{
		"<!doctype html>",
		"排产甘特图",
		"gantt-scroll",
		"M260727010046/客户&lt;甲&gt;&amp;",
		"class=\"bar late\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("HTML() missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, ">M260727010046/客户<甲>&<") {
		t.Fatalf("HTML() did not escape order label: %s", got)
	}
	if strings.Contains(got, "bar-label") {
		t.Fatalf("HTML() must not write labels inside schedule bars: %s", got)
	}
}

func TestSVGKeepsLabelsOutsideScheduleBars(t *testing.T) {
	payload := map[string]any{
		"plan": map[string]any{
			"items": []any{
				map[string]any{
					"teamName":  "A组",
					"styleNo":   "M260727010046",
					"startDate": "2026-08-03",
					"endDate":   "2026-08-03",
				},
			},
		},
	}

	got := SVG(payload)
	if strings.Contains(got, `fill="#ffffff"`) {
		t.Fatalf("SVG() still renders text inside bars: %s", got)
	}
	for _, want := range []string{"<svg", "M260727010046", "width=\"960\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("SVG() missing %q: %s", want, got)
		}
	}
}
