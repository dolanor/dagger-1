package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/mitchellh/colorstring"
	"github.com/rs/zerolog"
)

var colorize = colorstring.Colorize{
	Colors: colorstring.DefaultColors,
	Reset:  true,
}

type PlainOutput struct {
	Out io.Writer
}

const systemGroup = "system"

func (c *PlainOutput) Write(p []byte) (int, error) {
	event := map[string]interface{}{}
	d := json.NewDecoder(bytes.NewReader(p))
	if err := d.Decode(&event); err != nil {
		return 0, fmt.Errorf("cannot decode event: %s", err)
	}

	source := parseSource(event)

	fmt.Fprintln(c.Out, colorize.Color(fmt.Sprintf("%s %s %s%s%s",
		formatTimestamp(event),
		formatLevel(event),
		formatSource(source),
		formatMessage(event),
		formatFields(event),
	)))

	return len(p), nil
}

func formatLevel(event map[string]interface{}) string {
	level := zerolog.DebugLevel
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		level, _ = zerolog.ParseLevel(l)
	}

	fmtField := func(name, color string) string {
		return fmt.Sprintf("[%s]%-5s[reset]", color, strings.ToUpper(name))
	}

	switch level {
	case zerolog.TraceLevel:
		return fmtField("trace", "magenta")
	case zerolog.DebugLevel:
		return fmtField("debug", "yellow")
	case zerolog.InfoLevel:
		return fmtField("info", "green")
	case zerolog.WarnLevel:
		return fmtField("warn", "red")
	case zerolog.ErrorLevel:
		return fmtField("error", "red")
	case zerolog.FatalLevel:
		return fmtField("fatal", "red")
	case zerolog.PanicLevel:
		return fmtField("panic", "red")
	default:
		return fmtField("???", "bold")
	}
}

func formatLevelTerm(event map[string]interface{}) *logElem {
	level := zerolog.DebugLevel
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		level, _ = zerolog.ParseLevel(l)
	}

	switch level {
	case zerolog.TraceLevel:
		return &logElem{"TRC", "magenta", 0, nil}
	case zerolog.DebugLevel:
		return &logElem{"DBG", "yellow", 0, nil}
	case zerolog.InfoLevel:
		return &logElem{"INF", "green", 0, nil}
	case zerolog.WarnLevel:
		return &logElem{"WRN", "red", 0, nil}
	case zerolog.ErrorLevel:
		return &logElem{"ERR", "red", 0, nil}
	case zerolog.FatalLevel:
		return &logElem{"FTL", "red", 0, nil}
	case zerolog.PanicLevel:
		return &logElem{"PNC", "red", 0, nil}
	default:
		return &logElem{"???", "bold", 0, nil}
	}
}

func formatTimestamp(event map[string]interface{}) string {
	ts, ok := event[zerolog.TimestampFieldName].(string)
	if !ok {
		return "???"
	}

	t, err := time.Parse(zerolog.TimeFieldFormat, ts)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("[dark_gray]%s[reset]", t.Format(time.Kitchen))
}

func formatTimestampTerm(event map[string]interface{}) *logElem {
	ts, ok := event[zerolog.TimestampFieldName].(string)
	if !ok {
		return &logElem{"???", "", 0, nil}
	}

	t, err := time.Parse(zerolog.TimeFieldFormat, ts)
	if err != nil {
		panic(err)
	}
	return &logElem{t.Format(time.Kitchen), "dark_gray", 0, nil}
}

func formatMessage(event map[string]interface{}) string {
	message, ok := event[zerolog.MessageFieldName].(string)
	if !ok {
		return ""
	}
	message = strings.TrimSpace(message)

	if err, ok := event[zerolog.ErrorFieldName].(string); ok && err != "" {
		message = message + ": " + err
	}

	level := zerolog.DebugLevel
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		level, _ = zerolog.ParseLevel(l)
	}

	switch level {
	case zerolog.TraceLevel:
		return fmt.Sprintf("[dim]%s[reset]", message)
	case zerolog.DebugLevel:
		return fmt.Sprintf("[dim]%s[reset]", message)
	case zerolog.InfoLevel:
		return message
	case zerolog.WarnLevel:
		return fmt.Sprintf("[yellow]%s[reset]", message)
	case zerolog.ErrorLevel:
		return fmt.Sprintf("[red]%s[reset]", message)
	case zerolog.FatalLevel:
		return fmt.Sprintf("[red]%s[reset]", message)
	case zerolog.PanicLevel:
		return fmt.Sprintf("[red]%s[reset]", message)
	default:
		return message
	}
}

func formatMessageTerm(event map[string]interface{}) *logElem {
	message, ok := event[zerolog.MessageFieldName].(string)
	if !ok {
		return &logElem{"", "", 0, nil}
	}
	message = strings.TrimSpace(message)

	if err, ok := event[zerolog.ErrorFieldName].(string); ok && err != "" {
		message = message + ": " + err
	}

	level := zerolog.DebugLevel
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		level, _ = zerolog.ParseLevel(l)
	}

	switch level {
	case zerolog.TraceLevel:
		return &logElem{message, "dim", 0, nil}
	case zerolog.DebugLevel:
		return &logElem{message, "dim", 0, nil}
	case zerolog.InfoLevel:
		return &logElem{message, "", 0, nil}
	case zerolog.WarnLevel:
		return &logElem{message, "yellow", 0, nil}
	case zerolog.ErrorLevel:
		return &logElem{message, "red", 0, nil}
	case zerolog.FatalLevel:
		return &logElem{message, "red", 0, nil}
	case zerolog.PanicLevel:
		return &logElem{message, "red", 0, nil}
	default:
		return &logElem{message, "", 0, nil}
	}
}

func parseSource(event map[string]interface{}) string {
	source := systemGroup
	if task, ok := event["task"].(string); ok && task != "" {
		source = task
	}
	return source
}

func formatSource(source string) string {
	return fmt.Sprintf("[%s]%s | [reset]",
		hashColor(source),
		source,
	)
}

func formatFields(entry map[string]interface{}) string {
	// these are the fields we don't want to expose, either because they're
	// already part of the Log structure or because they're internal
	fieldSkipList := map[string]struct{}{
		zerolog.MessageFieldName:   {},
		zerolog.LevelFieldName:     {},
		zerolog.TimestampFieldName: {},
		zerolog.ErrorFieldName:     {},
		zerolog.CallerFieldName:    {},
		"environment":              {},
		"task":                     {},
		"state":                    {},
	}

	fields := []string{}
	for key, value := range entry {
		if _, ok := fieldSkipList[key]; ok {
			continue
		}
		switch v := value.(type) {
		case string:
			fields = append(fields, fmt.Sprintf("%s=%s", key, v))
		case int:
			fields = append(fields, fmt.Sprintf("%s=%v", key, v))
		case float64:
			dur := time.Duration(v) * time.Millisecond
			s := dur.Round(100 * time.Millisecond).String()
			fields = append(fields, fmt.Sprintf("%s=%s", key, s))
		case nil:
			fields = append(fields, fmt.Sprintf("%s=null", key))
		default:
			o, err := json.MarshalIndent(v, "", "    ")
			if err != nil {
				panic(err)
			}
			fields = append(fields, fmt.Sprintf("%s=%s", key, o))
		}
	}

	if len(fields) == 0 {
		return ""
	}
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i] < fields[j]
	})
	return fmt.Sprintf("    [bold]%s[reset]", strings.Join(fields, " "))
}

func formatFieldsTerm(entry map[string]interface{}) *logElem {
	// these are the fields we don't want to expose, either because they're
	// already part of the Log structure or because they're internal
	fieldSkipList := map[string]struct{}{
		zerolog.MessageFieldName:   {},
		zerolog.LevelFieldName:     {},
		zerolog.TimestampFieldName: {},
		zerolog.ErrorFieldName:     {},
		zerolog.CallerFieldName:    {},
		"environment":              {},
		"task":                     {},
		"state":                    {},
	}

	fields := []string{}
	for key, value := range entry {
		if _, ok := fieldSkipList[key]; ok {
			continue
		}
		switch v := value.(type) {
		case string:
			fields = append(fields, fmt.Sprintf("%s=%s", key, v))
		case int:
			fields = append(fields, fmt.Sprintf("%s=%v", key, v))
		case float64:
			dur := time.Duration(v) * time.Millisecond
			s := dur.Round(100 * time.Millisecond).String()
			fields = append(fields, fmt.Sprintf("%s=%s", key, s))
		case nil:
			fields = append(fields, fmt.Sprintf("%s=null", key))
		default:
			o, err := json.MarshalIndent(v, "", "    ")
			if err != nil {
				panic(err)
			}
			fields = append(fields, fmt.Sprintf("%s=%s", key, o))
		}
	}

	if len(fields) == 0 {
		return &logElem{"", "", 0, nil}
	}
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i] < fields[j]
	})
	return &logElem{"    " + strings.Join(fields, " "), "bold", 0, nil}
}

// hashColor returns a consistent color for a given string
func hashColor(text string) string {
	colors := []string{
		"green",
		"light_green",
		"light_blue",
		"blue",
		"magenta",
		"light_magenta",
		"light_yellow",
		"cyan",
		"light_cyan",
	}
	h := adler32.Checksum([]byte(text))
	return colors[int(h)%len(colors)]
}
