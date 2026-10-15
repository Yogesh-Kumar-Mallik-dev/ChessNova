package observability

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

type Logger struct {
	instanceID string
	mu         sync.Mutex
	isJSON     bool
	buffer     []LogEntry
	maxBuffer  int
}

var DefaultLogger = NewLogger("default", true)

func NewLogger(instanceID string, jsonFormat bool) *Logger {
	return &Logger{
		instanceID: instanceID,
		isJSON:     jsonFormat,
		buffer:     make([]LogEntry, 0, 500),
		maxBuffer:  500,
	}
}

func (l *Logger) SetInstanceID(id string) {
	l.instanceID = id
}

type LogEntry struct {
	Timestamp  string                 `json:"timestamp"`
	Level      string                 `json:"level"`
	InstanceID string                 `json:"instanceId"`
	Message    string                 `json:"message"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	now := time.Now().UTC().Format(time.RFC3339)
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp:  now,
		Level:      string(level),
		InstanceID: l.instanceID,
		Message:    msg,
		Fields:     fields,
	}

	// Append to in-memory circular buffer for observability
	l.buffer = append(l.buffer, entry)
	if len(l.buffer) > l.maxBuffer {
		l.buffer = l.buffer[len(l.buffer)-l.maxBuffer:]
	}

	if l.isJSON {
		data, err := json.Marshal(entry)
		if err == nil {
			fmt.Println(string(data))
			return
		}
	}

	// Plain text format
	if len(fields) > 0 {
		fmt.Printf("[%s] [%s] [%s] %s %v\n", now, level, l.instanceID, msg, fields)
	} else {
		fmt.Printf("[%s] [%s] [%s] %s\n", now, level, l.instanceID, msg)
	}
}

func (l *Logger) RecentLogs(limit int, levelFilter Level) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	var result []LogEntry
	for i := len(l.buffer) - 1; i >= 0; i-- {
		entry := l.buffer[i]
		if levelFilter != "" && !strings.EqualFold(entry.Level, string(levelFilter)) {
			continue
		}
		result = append(result, entry)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func (l *Logger) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		lvl := Level(strings.ToUpper(r.URL.Query().Get("level")))
		logs := l.RecentLogs(limit, lvl)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"instanceId": l.instanceID,
			"count":      len(logs),
			"logs":       logs,
		})
	}
}

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelDebug, msg, f)
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelInfo, msg, f)
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelWarn, msg, f)
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelError, msg, f)
}

// statusWriter wraps http.ResponseWriter to intercept response status code and bytes written
type statusWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *statusWriter) WriteHeader(status int) {
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("http.Hijacker not supported by underlying ResponseWriter")
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// HTTPLoggingMiddleware creates an HTTP middleware that records metrics and emits structured logs
func HTTPLoggingMiddleware(logger *Logger, metrics *MetricRegistry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || strings.HasPrefix(r.URL.Path, "/ws/") {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		sw := &statusWriter{
			ResponseWriter: w,
			statusCode:     0,
		}

		next.ServeHTTP(sw, r)

		if sw.statusCode == 0 {
			sw.statusCode = http.StatusOK
		}

		duration := time.Since(start)
		durationMs := duration.Milliseconds()

		// Record in metrics registry
		if metrics != nil {
			metrics.RecordHTTPRequest(r.Method, r.URL.Path, sw.statusCode, durationMs)
		}

		// Skip log flooding on healthcheck unless it fails
		if r.URL.Path == "/health" && sw.statusCode < 400 {
			return
		}

		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.Header.Get("X-Real-IP")
		}
		if clientIP == "" {
			clientIP = strings.Split(r.RemoteAddr, ":")[0]
		}

		fields := map[string]interface{}{
			"method":       r.Method,
			"path":         r.URL.Path,
			"status":       sw.statusCode,
			"durationMs":   durationMs,
			"clientIp":     clientIP,
			"bytesWritten": sw.bytesWritten,
		}

		lvl := LevelInfo
		if sw.statusCode >= 500 {
			lvl = LevelError
		} else if sw.statusCode >= 400 {
			lvl = LevelWarn
		}

		if logger != nil {
			logger.log(lvl, fmt.Sprintf("%s %s -> %d (%dms)", r.Method, r.URL.Path, sw.statusCode, durationMs), fields)
		} else {
			_ = os.Stdout
		}
	})
}
