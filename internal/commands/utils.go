package commands

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

func ParseGuildID(guildID string) int64 {
	id, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		log.Printf("Failed to parse guild ID '%s': %v", guildID, err)
		return 0
	}
	return id
}

func boolPtr(b bool) *bool {
	return &b
}

// ParseMentionIDs extracts user IDs from mentions and raw ID strings.
func ParseMentionIDs(text string) []string {
	// Supports <@123>, <@!123>, and raw IDs separated by spaces
	re := regexp.MustCompile(`<@!?([0-9]+)>`)
	var ids []string
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		if len(m) >= 2 {
			ids = append(ids, m[1])
		}
	}
	// also allow raw IDs separated by spaces
	for _, tok := range strings.Fields(text) {
		if tok == "" {
			continue
		}
		// if it's pure digits, treat as ID
		if AllDigits(tok) {
			ids = append(ids, tok)
		}
	}
	return Unique(ids)
}

func AllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

func Unique(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func ParseDHMToMinutes(input string) (int, error) {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return 1440, nil
	}
	if AllDigits(s) {
		// backward-compatible: treat pure digits as minutes
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("interval の解析に失敗しました: %w", err)
		}
		return int(v), nil
	}

	re := regexp.MustCompile(`(?i)(\d+)([dhm])`)
	matches := re.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("interval は 1d2h3m の形式で指定してください (例: 1d / 2h / 30m / 1d2h3m)")
	}

	var total int64
	pos := 0
	for _, m := range matches {
		if m[0] != pos {
			return 0, fmt.Errorf("interval は 1d2h3m の形式で指定してください (例: 1d2h3m)")
		}
		numStr := s[m[2]:m[3]]
		unit := s[m[4]:m[5]]
		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("interval の解析に失敗しました: %w", err)
		}
		switch unit {
		case "d":
			total += n * 24 * 60
		case "h":
			total += n * 60
		case "m":
			total += n
		default:
			return 0, fmt.Errorf("interval は d/h/m のみ対応です")
		}
		pos = m[1]
	}
	if pos != len(s) {
		return 0, fmt.Errorf("interval は 1d2h3m の形式で指定してください (例: 1d2h3m)")
	}
	if total > int64(^uint(0)>>1) {
		return 0, fmt.Errorf("interval が大きすぎます")
	}
	return int(total), nil
}
