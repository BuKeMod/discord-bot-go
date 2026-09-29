package utility_party

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	timeHHMMRegex = regexp.MustCompile(`^(\d{1,2})[:.](\d{2})$`)
	durationRegex = regexp.MustCompile(`^(?:อีก\s*)?(\d+)\s*(?:m|min|นาที)$`)
)

// ParseGameTime parses various time inputs into a user-friendly display string and unix timestamp.
func ParseGameTime(input string) (string, int64) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "⚡ ตอนนี้เลย / เมื่อคนครบ", 0
	}

	lower := strings.ToLower(trimmed)
	if lower == "now" || lower == "ตอนนี้" || lower == "ทันที" || lower == "ด่วน" {
		now := time.Now().Unix()
		return "⚡ ตอนนี้เลย (ทันทีเมื่อคนครบ)", now
	}

	// Case 1: Duration like "30m", "30 นาที", "อีก 15 นาที"
	if m := durationRegex.FindStringSubmatch(lower); m != nil {
		mins, _ := strconv.Atoi(m[1])
		if mins > 0 && mins <= 1440 {
			target := time.Now().Add(time.Duration(mins) * time.Minute)
			ts := target.Unix()
			return fmt.Sprintf("ในอีก %d นาที (<t:%d:t> • <t:%d:R>)", mins, ts, ts), ts
		}
	}

	// Case 2: Specific clock time like "21:00", "20.30"
	if m := timeHHMMRegex.FindStringSubmatch(trimmed); m != nil {
		hr, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		if hr >= 0 && hr < 24 && min >= 0 && min < 60 {
			loc := time.FixedZone("ICT", 7*3600) // Thailand UTC+7
			now := time.Now().In(loc)
			target := time.Date(now.Year(), now.Month(), now.Day(), hr, min, 0, 0, loc)
			if target.Before(now.Add(-5 * time.Minute)) {
				// Time already passed today, assume tomorrow
				target = target.Add(24 * time.Hour)
			}
			ts := target.Unix()
			return fmt.Sprintf("%02d:%02d น. (<t:%d:t> • <t:%d:R>)", hr, min, ts, ts), ts
		}
	}

	// Fallback to verbatim user string
	return trimmed, 0
}
