package utils

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type LogrusColorFormatter struct{}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

const (
	reset = "\033[0m"

	bgRed    = "\033[41m"
	bgGreen  = "\033[42m"
	bgYellow = "\033[43m"
	bgBlue   = "\033[44m"
	bgCyan   = "\033[46m"

	// Optional: make text bold & white for contrast
	whiteText = "\033[97m"
	bold      = "\033[1m"
)

func (f *LogrusColorFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b bytes.Buffer

	timestamp := entry.Time.Format(time.RFC3339)

	// Level color
	levelColor := colorBlue
	if entry.Level == logrus.WarnLevel {
		levelColor = colorYellow
	} else if entry.Level == logrus.ErrorLevel || entry.Level == logrus.FatalLevel {
		levelColor = colorRed
	}

	// Base log
	b.WriteString(fmt.Sprintf(
		`time="%s" level=%s%s%s msg="%s"`,
		timestamp,
		levelColor, strings.ToUpper(entry.Level.String()), colorReset,
		entry.Message,
	))

	// Custom fields with background blocks
	for k, v := range entry.Data {
		switch k {

		case "method":
			b.WriteString(fmt.Sprintf(" %s=%s%v%s", k, colorCyan, v, colorReset))

		case "path":
			b.WriteString(fmt.Sprintf(" %s=%s%v%s", k, colorBlue, v, colorReset))

		case "latency":
			b.WriteString(fmt.Sprintf(" %s=%s%v%s", k, colorYellow, v, colorReset))
			// b.WriteString(fmt.Sprintf(" %s%s %s=%v %s",
			// 	bgYellow, whiteText, k, v, reset))

		case "error":
			b.WriteString(fmt.Sprintf(" %s=%s%v%s", k, colorRed, v, colorReset))

		case "status":
			bg := bgGreen
			if status, ok := v.(int); ok && status >= 400 {
				bg = bgRed
			}

			b.WriteString(fmt.Sprintf(" %s%s %s=%v %s",
				bg, whiteText, k, v, reset))

		case "event", "ws_action":
			var bg string
			eventVal := fmt.Sprintf("%v", v)
			switch eventVal {
			case "CONNECT", "ws_connect":
				bg = bgGreen // Green for connections
			case "DISCONNECT", "ws_disconnect":
				bg = bgYellow // Yellow for disconnections
			case "ERROR", "ws_error":
				bg = bgRed // Red for errors
			case "BROADCAST", "ws_broadcast":
				bg = bgBlue // Blue for broadcasts
			default:
				bg = bgCyan // Cyan for unknown events
			}
			b.WriteString(fmt.Sprintf(" %s%s %s=%v %s",
				bg, whiteText, k, v, reset))

		default:
			b.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}
