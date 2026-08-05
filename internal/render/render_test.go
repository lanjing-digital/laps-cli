package render

import (
	"strings"
	"testing"
)

func TestHTMLRendersScrollableGanttWithInteractiveDetailTooltip(t *testing.T) {
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
		`id="schedule-tooltip"`,
		`data-tooltip="班组：A组`,
		"pointerenter",
		"tabindex=\"0\"",
		"交期状态：可能逾期",
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
	if strings.Contains(got, `class="bar late" title=`) {
		t.Fatalf("HTML() must use the in-page tooltip instead of a browser title: %s", got)
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

func TestHTMLRendersCandidateComparisonUsingOfficialGantt(t *testing.T) {
	item := map[string]any{"teamName": "A组", "styleNo": "FG-01", "startDate": "2026-08-03", "endDate": "2026-08-05"}
	payload := map[string]any{"plan": map[string]any{
		"items": []any{item},
		"optimization": map[string]any{
			"mode": "portfolio", "selectedSolver": "heuristic", "recommendedSolver": "cp-sat", "shadow": true,
			"candidates": []any{
				map[string]any{"solver": "heuristic", "status": "feasible", "durationMs": 10, "metrics": map[string]any{}, "plan": map[string]any{"items": []any{item}}},
				map[string]any{"solver": "cp-sat", "status": "optimal", "durationMs": 20, "metrics": map[string]any{}, "plan": map[string]any{"items": []any{item}}},
			},
		},
	}}
	got := HTML(payload)
	for _, want := range []string{"排产方案对比", "当前采用", "推荐", "iframe", "gantt-scroll", "08-03"} {
		if !strings.Contains(got, want) {
			t.Fatalf("HTML() missing %q: %s", want, got)
		}
	}
}
