package genspark

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type Event struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	MessageID string `json:"message_id"`
	FieldName string `json:"field_name"`
	FieldVal  string `json:"field_value"`
	Delta     string `json:"delta"`
	Content   string `json:"content"`
	Role      string `json:"role"`
	Action    any    `json:"action"`
}

func ParseBodyAsEvents(body string) ([]Event, error) {
	reader := strings.NewReader(body)
	scanner := bufio.NewScanner(reader)
	out := make([]Event, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		}
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.New("no events")
	}
	return out, nil
}

func ReadSSELines(r io.Reader, fn func(rawLine string) error) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if err := fn(line); err != nil {
			return err
		}
	}
	return scanner.Err()
}
