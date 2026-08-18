package repoutil

import (
	"encoding/json"
	"fmt"
	"time"
)

const TimeLayout = time.RFC3339Nano

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(TimeLayout)
}

func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(TimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp: %w", err)
	}
	return t.UTC(), nil
}

func MarshalMap(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	data, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func UnmarshalMap(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("decode string map: %w", err)
	}
	return out, nil
}

func MarshalStrings(s []string) string {
	if len(s) == 0 {
		return "[]"
	}
	data, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func UnmarshalStrings(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("decode string slice: %w", err)
	}
	return out, nil
}
