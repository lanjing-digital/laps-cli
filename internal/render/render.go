package render

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"
	"time"
)

type Block struct {
	TeamName        string
	OrderLabel      string
	StartDate       time.Time
	EndDate         time.Time
	AllocatedQty    float64
	Efficiency      float64
	CanMeetDeadline *bool
}

type teamGroup struct {
	name   string
	blocks []Block
}

func Timeline(payload map[string]any) string {
	blocks := ExtractBlocks(payload)
	if len(blocks) == 0 {
		return "No schedule blocks with start/end dates found in response.\n"
	}

	minDate, maxDate := dateRange(blocks)
	totalDays := max(1, daysBetween(minDate, maxDate)+1)
	scale := max(1, int(math.Ceil(float64(totalDays)/80.0)))
	groups := groupBlocks(blocks)

	var b strings.Builder
	fmt.Fprintf(&b, "Schedule timeline\n")
	fmt.Fprintf(&b, "Range: %s -> %s  (%d char = %d day)\n", formatDate(minDate), formatDate(maxDate), 1, scale)
	fmt.Fprintf(&b, "\n")
	for _, group := range groups {
		fmt.Fprintf(&b, "%s\n", group.name)
		for _, block := range group.blocks {
			offset := daysBetween(minDate, block.StartDate) / scale
			duration := max(1, daysBetween(block.StartDate, block.EndDate)+1)
			width := max(3, int(math.Ceil(float64(duration)/float64(scale))))
			fmt.Fprintf(
				&b,
				"  | %s[%s] %s  start=%s  end=%s  qty=%s  eff=%s%s\n",
				strings.Repeat(" ", max(0, offset)),
				strings.Repeat("=", width),
				block.OrderLabel,
				formatDate(block.StartDate),
				formatDate(block.EndDate),
				formatNumber(block.AllocatedQty),
				formatPercent(block.Efficiency),
				deadlineSuffix(block.CanMeetDeadline),
			)
		}
		fmt.Fprintf(&b, "\n")
	}
	return b.String()
}

func SVG(payload map[string]any) string {
	blocks := ExtractBlocks(payload)
	if len(blocks) == 0 {
		return `<svg xmlns="http://www.w3.org/2000/svg" width="720" height="120" viewBox="0 0 720 120"><text x="24" y="64" font-family="Arial, sans-serif" font-size="16" fill="#1f2937">No schedule blocks with start/end dates found in response.</text></svg>` + "\n"
	}

	minDate, maxDate := dateRange(blocks)
	totalDays := max(1, daysBetween(minDate, maxDate)+1)
	groups := groupBlocks(blocks)
	rowCount := 0
	for _, group := range groups {
		rowCount += len(group.blocks) + 1
	}

	left := 240
	top := 86
	dayWidth := 16
	if totalDays > 60 {
		dayWidth = 12
	}
	if totalDays > 100 {
		dayWidth = 8
	}
	width := max(960, left+totalDays*dayWidth+48)
	height := top + rowCount*34 + 48

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", width, height, width, height)
	fmt.Fprintf(&b, `<rect width="100%%" height="100%%" fill="#f8fafc"/>`+"\n")
	fmt.Fprintf(&b, `<text x="24" y="30" font-family="Arial, sans-serif" font-size="18" font-weight="700" fill="#0f172a">Schedule Gantt</text>`+"\n")
	fmt.Fprintf(&b, `<text x="24" y="52" font-family="Arial, sans-serif" font-size="12" fill="#475569">%s -> %s · %d blocks</text>`+"\n", formatDate(minDate), formatDate(maxDate), len(blocks))

	for day := 0; day < totalDays; day += max(1, int(math.Ceil(float64(totalDays)/12.0))) {
		x := left + day*dayWidth
		date := minDate.AddDate(0, 0, day)
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#cbd5e1" stroke-width="1"/>`+"\n", x, top-10, x, height-24)
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-family="Arial, sans-serif" font-size="10" fill="#64748b">%s</text>`+"\n", x+2, top-20, html.EscapeString(date.Format("01-02")))
	}

	y := top
	colorIndex := 0
	colors := []string{"#2563eb", "#059669", "#d97706", "#7c3aed", "#dc2626", "#0891b2"}
	for _, group := range groups {
		fmt.Fprintf(&b, `<text x="24" y="%d" font-family="Arial, sans-serif" font-size="13" font-weight="700" fill="#0f172a">%s</text>`+"\n", y+18, html.EscapeString(group.name))
		y += 32
		for _, block := range group.blocks {
			x := left + daysBetween(minDate, block.StartDate)*dayWidth
			w := max(28, (daysBetween(block.StartDate, block.EndDate)+1)*dayWidth)
			color := colors[colorIndex%len(colors)]
			colorIndex += 1
			label := fmt.Sprintf("%s  %s -> %s", block.OrderLabel, formatDate(block.StartDate), formatDate(block.EndDate))
			fmt.Fprintf(&b, `<text x="42" y="%d" font-family="Arial, sans-serif" font-size="11" fill="#334155">%s</text>`+"\n", y+16, html.EscapeString(shorten(block.OrderLabel, 20)))
			fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="22" rx="4" fill="%s" opacity="0.88"/>`+"\n", x, y, w, color)
			fmt.Fprintf(&b, `<title>%s</title>`+"\n", html.EscapeString(label))
			y += 34
		}
	}
	fmt.Fprintf(&b, "</svg>\n")
	return b.String()
}

// HTML returns a self-contained, scrollable Gantt view that is safe to embed in
// a sandboxed iframe. Labels deliberately stay outside schedule bars so short
// allocations never produce overlapping text.
func HTML(payload map[string]any) string {
	blocks := ExtractBlocks(payload)
	if len(blocks) == 0 {
		return `<!doctype html><html lang="en"><meta charset="utf-8"><body><p>No schedule blocks with start/end dates found in response.</p></body></html>` + "\n"
	}

	minDate, maxDate := dateRange(blocks)
	totalDays := max(1, daysBetween(minDate, maxDate)+1)
	groups := groupBlocks(blocks)
	teamCount := len(groups)
	timelineWidth := max(860, totalDays*22)
	tickDays := dateTicks(totalDays)

	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>排产甘特图</title>
<style>
:root { color: #1e293b; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
* { box-sizing: border-box; }
body { margin: 0; background: #f8fafc; }
.page { min-width: 760px; padding: 20px; }
h1 { margin: 0; font-size: 20px; line-height: 1.3; color: #0f172a; }
.meta { margin: 6px 0 18px; color: #64748b; font-size: 13px; }
.gantt-scroll { overflow: auto; border: 1px solid #dbe3ef; border-radius: 10px; background: #fff; }
.gantt { min-width: var(--timeline-total-width); }
.axis, .gantt-row { display: grid; grid-template-columns: 220px var(--timeline-width); }
.axis { position: sticky; top: 0; z-index: 3; background: #f8fafc; border-bottom: 1px solid #dbe3ef; }
.axis-label { padding: 10px 14px; color: #64748b; font-size: 12px; font-weight: 600; }
.axis-track { position: relative; height: 40px; overflow: visible; }
.tick { position: absolute; top: 0; bottom: 0; border-left: 1px solid #dbe3ef; }
.tick-label { position: absolute; top: 11px; transform: translateX(-50%); color: #64748b; font-size: 11px; white-space: nowrap; }
.tick-label.first { transform: none; }
.tick-label.last { transform: translateX(-100%); }
.team-heading { position: sticky; left: 0; z-index: 2; display: grid; grid-template-columns: 220px var(--timeline-width); background: #f1f5f9; border-top: 1px solid #dbe3ef; border-bottom: 1px solid #dbe3ef; }
.team-heading span { padding: 8px 14px; color: #334155; font-size: 13px; font-weight: 700; }
.gantt-row { min-height: 34px; background: #fff; }
.order-label { position: sticky; left: 0; z-index: 1; display: flex; align-items: center; padding: 6px 14px; overflow: hidden; border-right: 1px solid #e2e8f0; border-bottom: 1px solid #eef2f7; background: #fff; color: #334155; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.track { position: relative; min-height: 34px; border-bottom: 1px solid #eef2f7; background-image: repeating-linear-gradient(to right, transparent 0, transparent calc(12.5% - 1px), #eef2f7 calc(12.5% - 1px), #eef2f7 12.5%); }
.bar { position: absolute; top: 7px; height: 20px; min-width: 5px; border-radius: 4px; cursor: help; box-shadow: inset 0 0 0 1px rgb(15 23 42 / 12%); }
.bar.late { outline: 2px solid #dc2626; outline-offset: 1px; }
.bar:focus-visible { outline: 3px solid #0f172a; outline-offset: 2px; }
.schedule-tooltip { position: fixed; z-index: 20; max-width: 340px; padding: 9px 11px; border: 1px solid #cbd5e1; border-radius: 8px; background: rgb(15 23 42 / 96%); color: #f8fafc; font-size: 12px; line-height: 1.55; white-space: pre-line; pointer-events: none; opacity: 0; transform: translateY(4px); transition: opacity 120ms ease-out, transform 120ms ease-out; }
.schedule-tooltip.visible { opacity: 1; transform: translateY(0); }
.legend { display: flex; flex-wrap: wrap; gap: 12px 18px; margin-top: 14px; color: #475569; font-size: 12px; }
.legend-item { display: inline-flex; align-items: center; gap: 6px; }
.legend-swatch { width: 12px; height: 12px; border-radius: 3px; background: #2563eb; }
.legend-swatch.late { background: #fff; border: 2px solid #dc2626; }
</style>
</head>
<body>
`)
	fmt.Fprintf(&b, `<main class="page" style="--timeline-width: %dpx; --timeline-total-width: %dpx">`, timelineWidth, timelineWidth+220)
	fmt.Fprintf(&b, `<h1>排产甘特图</h1><p class="meta">%s 至 %s · %d 个排产区块 · %d 个班组</p>`, html.EscapeString(formatDate(minDate)), html.EscapeString(formatDate(maxDate)), len(blocks), teamCount)
	b.WriteString(`<div class="gantt-scroll"><div class="gantt"><div class="axis"><div class="axis-label">订单 / 款号</div><div class="axis-track">`)
	for index, day := range tickDays {
		left := float64(day) / float64(totalDays) * 100
		className := "tick-label"
		if index == 0 {
			className += " first"
		} else if index == len(tickDays)-1 {
			className += " last"
		}
		date := minDate.AddDate(0, 0, day).Format("01-02")
		fmt.Fprintf(&b, `<span class="tick" style="left: %.4f%%"></span><span class="%s" style="left: %.4f%%">%s</span>`, left, className, left, html.EscapeString(date))
	}
	b.WriteString(`</div></div>`)

	colors := []string{"#2563eb", "#059669", "#d97706", "#7c3aed", "#0891b2", "#db2777"}
	for groupIndex, group := range groups {
		fmt.Fprintf(&b, `<section><div class="team-heading"><span>%s</span><span></span></div>`, html.EscapeString(group.name))
		for _, block := range group.blocks {
			start := daysBetween(minDate, block.StartDate)
			duration := max(1, daysBetween(block.StartDate, block.EndDate)+1)
			left := float64(start) / float64(totalDays) * 100
			width := float64(duration) / float64(totalDays) * 100
			className := "bar"
			if block.CanMeetDeadline != nil && !*block.CanMeetDeadline {
				className += " late"
			}
			details := tooltipDetails(group.name, block)
			fmt.Fprintf(&b, `<div class="gantt-row"><div class="order-label" title="%s">%s</div><div class="track"><div class="%s" data-tooltip="%s" aria-label="%s" aria-describedby="schedule-tooltip" role="img" tabindex="0" style="left: %.4f%%; width: %.4f%%; background: %s"></div></div></div>`, html.EscapeString(details), html.EscapeString(shorten(block.OrderLabel, 34)), className, html.EscapeString(details), html.EscapeString(details), left, width, colors[groupIndex%len(colors)])
		}
		b.WriteString(`</section>`)
	}
	b.WriteString(`</div></div><div class="legend"><span class="legend-item"><i class="legend-swatch"></i>排产区块（悬停或聚焦查看详细信息）</span><span class="legend-item"><i class="legend-swatch late"></i>可能逾期</span></div></main><div id="schedule-tooltip" class="schedule-tooltip" role="tooltip" aria-hidden="true"></div><script>
(() => {
  const tooltip = document.getElementById("schedule-tooltip");
  let activeBar = null;
  const place = (bar, event) => {
    const rect = bar.getBoundingClientRect();
    const pointX = event && Number.isFinite(event.clientX) ? event.clientX : rect.left + rect.width / 2;
    const pointY = event && Number.isFinite(event.clientY) ? event.clientY : rect.bottom;
    tooltip.style.left = "0px";
    tooltip.style.top = "0px";
    const bounds = tooltip.getBoundingClientRect();
    const left = Math.max(8, Math.min(pointX + 14, window.innerWidth - bounds.width - 8));
    const top = Math.max(8, Math.min(pointY + 14, window.innerHeight - bounds.height - 8));
    tooltip.style.left = left + "px";
    tooltip.style.top = top + "px";
  };
  const show = (bar, event) => {
    activeBar = bar;
    tooltip.textContent = bar.dataset.tooltip || bar.getAttribute("aria-label") || "";
    tooltip.setAttribute("aria-hidden", "false");
    tooltip.classList.add("visible");
    place(bar, event);
  };
  const hide = (bar) => {
    if (bar && activeBar !== bar) return;
    activeBar = null;
    tooltip.classList.remove("visible");
    tooltip.setAttribute("aria-hidden", "true");
  };
  document.querySelectorAll(".bar[data-tooltip]").forEach((bar) => {
    bar.addEventListener("pointerenter", (event) => show(bar, event));
    bar.addEventListener("pointermove", (event) => show(bar, event));
    bar.addEventListener("pointerleave", () => hide(bar));
    bar.addEventListener("focus", () => show(bar));
    bar.addEventListener("blur", () => hide(bar));
    bar.addEventListener("keydown", (event) => { if (event.key === "Escape") hide(bar); });
  });
  window.addEventListener("scroll", () => activeBar && place(activeBar), true);
  window.addEventListener("resize", () => activeBar && place(activeBar));
})();
</script></body></html>`)
	return b.String() + "\n"
}

func tooltipDetails(teamName string, block Block) string {
	deadline := "交期状态：待确认"
	if block.CanMeetDeadline != nil {
		if *block.CanMeetDeadline {
			deadline = "交期状态：可按期完成"
		} else {
			deadline = "交期状态：可能逾期"
		}
	}
	return fmt.Sprintf("班组：%s\n订单：%s\n计划：%s 至 %s\n分配数量：%s\n效率：%s\n%s", teamName, block.OrderLabel, formatDate(block.StartDate), formatDate(block.EndDate), formatNumber(block.AllocatedQty), formatPercent(block.Efficiency), deadline)
}

func dateTicks(totalDays int) []int {
	step := max(1, int(math.Ceil(float64(totalDays)/8.0)))
	ticks := make([]int, 0, 10)
	for day := 0; day < totalDays; day += step {
		ticks = append(ticks, day)
	}
	if ticks[len(ticks)-1] != totalDays-1 {
		ticks = append(ticks, totalDays-1)
	}
	return ticks
}

func ExtractBlocks(payload map[string]any) []Block {
	var blocks []Block
	if plan, ok := asMap(payload["plan"]); ok {
		if items, ok := asSlice(plan["items"]); ok {
			for _, raw := range items {
				if item, ok := asMap(raw); ok {
					if block, ok := blockFromMap(item, "teamName"); ok {
						blocks = append(blocks, block)
					}
				}
			}
		} else if block, ok := blockFromMap(plan, "targetTeamName"); ok {
			blocks = append(blocks, block)
		}
	}
	if records, ok := asSlice(payload["records"]); ok {
		for _, raw := range records {
			if record, ok := asMap(raw); ok {
				if block, ok := blockFromMap(record, "teamName"); ok {
					blocks = append(blocks, block)
				}
			}
		}
	}
	sortBlocks(blocks)
	return blocks
}

func blockFromMap(m map[string]any, teamKey string) (Block, bool) {
	start, ok := timeFromAny(m["startDate"])
	if !ok {
		return Block{}, false
	}
	end, ok := timeFromAny(m["endDate"])
	if !ok {
		return Block{}, false
	}
	label := firstString(m, "styleNo", "orderId", "scheduleId", "recordId", "id")
	if label == "" {
		label = "order"
	}
	customer := stringValue(m["customerName"])
	if customer != "" {
		label = label + "/" + customer
	}
	return Block{
		TeamName:        fallback(firstString(m, teamKey, "teamName", "targetTeamId", "teamId"), "Unknown team"),
		OrderLabel:      label,
		StartDate:       start,
		EndDate:         end,
		AllocatedQty:    numberValue(m["allocatedQty"]),
		Efficiency:      numberValue(m["efficiency"]),
		CanMeetDeadline: boolPointer(m["canMeetDeadline"]),
	}, true
}

func groupBlocks(blocks []Block) []teamGroup {
	byTeam := map[string][]Block{}
	for _, block := range blocks {
		byTeam[block.TeamName] = append(byTeam[block.TeamName], block)
	}
	names := make([]string, 0, len(byTeam))
	for name := range byTeam {
		names = append(names, name)
	}
	sort.Strings(names)
	groups := make([]teamGroup, 0, len(names))
	for _, name := range names {
		groupBlocks := byTeam[name]
		sortBlocks(groupBlocks)
		groups = append(groups, teamGroup{name: name, blocks: groupBlocks})
	}
	return groups
}

func sortBlocks(blocks []Block) {
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].TeamName != blocks[j].TeamName {
			return blocks[i].TeamName < blocks[j].TeamName
		}
		if !blocks[i].StartDate.Equal(blocks[j].StartDate) {
			return blocks[i].StartDate.Before(blocks[j].StartDate)
		}
		return blocks[i].OrderLabel < blocks[j].OrderLabel
	})
}

func dateRange(blocks []Block) (time.Time, time.Time) {
	minDate := truncateDay(blocks[0].StartDate)
	maxDate := truncateDay(blocks[0].EndDate)
	for _, block := range blocks[1:] {
		start := truncateDay(block.StartDate)
		end := truncateDay(block.EndDate)
		if start.Before(minDate) {
			minDate = start
		}
		if end.After(maxDate) {
			maxDate = end
		}
	}
	return minDate, maxDate
}

func timeFromAny(value any) (time.Time, bool) {
	switch v := value.(type) {
	case float64:
		if v <= 0 {
			return time.Time{}, false
		}
		return truncateDay(time.UnixMilli(int64(v))), true
	case int64:
		if v <= 0 {
			return time.Time{}, false
		}
		return truncateDay(time.UnixMilli(v)), true
	case int:
		if v <= 0 {
			return time.Time{}, false
		}
		return truncateDay(time.UnixMilli(int64(v))), true
	case string:
		if v == "" {
			return time.Time{}, false
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02"} {
			if parsed, err := time.Parse(layout, v); err == nil {
				return truncateDay(parsed), true
			}
		}
	}
	return time.Time{}, false
}

func asMap(value any) (map[string]any, bool) {
	m, ok := value.(map[string]any)
	return m, ok
}

func asSlice(value any) ([]any, bool) {
	s, ok := value.([]any)
	return s, ok
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(m[key]); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func numberValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func boolPointer(value any) *bool {
	v, ok := value.(bool)
	if !ok {
		return nil
	}
	return &v
}

func daysBetween(start time.Time, end time.Time) int {
	return int(truncateDay(end).Sub(truncateDay(start)).Hours() / 24)
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func formatNumber(value float64) string {
	if value == 0 {
		return "-"
	}
	if math.Abs(value-math.Round(value)) < 0.000001 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

func formatPercent(value float64) string {
	if value == 0 {
		return "-"
	}
	if value <= 1 {
		value *= 100
	}
	return fmt.Sprintf("%.1f%%", value)
}

func deadlineSuffix(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "  deadline=ok"
	}
	return "  deadline=late"
}

func fallback(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func shorten(value string, maxLen int) string {
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}
