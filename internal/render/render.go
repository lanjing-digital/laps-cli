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

	left := 180
	top := 56
	dayWidth := 18
	if totalDays > 60 {
		dayWidth = 12
	}
	if totalDays > 100 {
		dayWidth = 8
	}
	width := left + totalDays*dayWidth + 48
	height := top + rowCount*34 + 48

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", width, height, width, height)
	fmt.Fprintf(&b, `<rect width="100%%" height="100%%" fill="#f8fafc"/>`+"\n")
	fmt.Fprintf(&b, `<text x="24" y="30" font-family="Arial, sans-serif" font-size="18" font-weight="700" fill="#0f172a">Schedule Gantt</text>`+"\n")
	fmt.Fprintf(&b, `<text x="%d" y="30" font-family="Arial, sans-serif" font-size="12" fill="#475569">%s -> %s</text>`+"\n", left, formatDate(minDate), formatDate(maxDate))

	for day := 0; day < totalDays; day += max(1, int(math.Ceil(float64(totalDays)/12.0))) {
		x := left + day*dayWidth
		date := minDate.AddDate(0, 0, day)
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#cbd5e1" stroke-width="1"/>`+"\n", x, top-18, x, height-24)
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-family="Arial, sans-serif" font-size="10" fill="#64748b">%s</text>`+"\n", x+2, top-24, html.EscapeString(date.Format("01-02")))
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
			fmt.Fprintf(&b, `<text x="%d" y="%d" font-family="Arial, sans-serif" font-size="10" fill="#ffffff">%s</text>`+"\n", x+6, y+15, html.EscapeString(shorten(label, max(8, w/6))))
			y += 34
		}
	}
	fmt.Fprintf(&b, "</svg>\n")
	return b.String()
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
