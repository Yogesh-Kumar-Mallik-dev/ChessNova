package observability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MetricRegistry manages in-memory performance and application metrics
type MetricRegistry struct {
	instanceID string
	startTime  time.Time

	// Counters
	httpRequestsTotal   map[string]*int64
	httpRequestsMu      sync.RWMutex
	httpDurationMsSum   map[string]*int64
	httpDurationMsCount map[string]*int64
	httpDurationMu      sync.RWMutex

	// Gauges
	activeWebSockets      int64
	activeGames           int64
	matchmakingQueueSize  int64
	matchmakingJoins      int64
	matchmakingLeaves     int64
	matchmakingMatches    int64
	stockfishEvaluations  int64
}

var DefaultMetrics = NewMetricRegistry("default")

func NewMetricRegistry(instanceID string) *MetricRegistry {
	return &MetricRegistry{
		instanceID:          instanceID,
		startTime:           time.Now().UTC(),
		httpRequestsTotal:   make(map[string]*int64),
		httpDurationMsSum:   make(map[string]*int64),
		httpDurationMsCount: make(map[string]*int64),
	}
}

func (m *MetricRegistry) SetInstanceID(id string) {
	m.instanceID = id
}

func (m *MetricRegistry) RecordHTTPRequest(method, path string, status int, durationMs int64) {
	// Normalize path to avoid high cardinality (replace variable IDs with param tags)
	normalizedPath := normalizePath(path)
	key := fmt.Sprintf("%s|%s|%d", method, normalizedPath, status)

	m.httpRequestsMu.Lock()
	ptr, exists := m.httpRequestsTotal[key]
	if !exists {
		var val int64
		ptr = &val
		m.httpRequestsTotal[key] = ptr
	}
	m.httpRequestsMu.Unlock()
	atomic.AddInt64(ptr, 1)

	// Record latency stats
	routeKey := fmt.Sprintf("%s|%s", method, normalizedPath)
	m.httpDurationMu.Lock()
	sPtr, sExists := m.httpDurationMsSum[routeKey]
	if !sExists {
		var sVal int64
		sPtr = &sVal
		m.httpDurationMsSum[routeKey] = sPtr
	}
	cPtr, cExists := m.httpDurationMsCount[routeKey]
	if !cExists {
		var cVal int64
		cPtr = &cVal
		m.httpDurationMsCount[routeKey] = cPtr
	}
	m.httpDurationMu.Unlock()

	atomic.AddInt64(sPtr, durationMs)
	atomic.AddInt64(cPtr, 1)
}

func (m *MetricRegistry) IncActiveWebSocket() {
	atomic.AddInt64(&m.activeWebSockets, 1)
}

func (m *MetricRegistry) DecActiveWebSocket() {
	atomic.AddInt64(&m.activeWebSockets, -1)
}

func (m *MetricRegistry) SetActiveGames(count int64) {
	atomic.StoreInt64(&m.activeGames, count)
}

func (m *MetricRegistry) SetMatchmakingQueueSize(size int64) {
	atomic.StoreInt64(&m.matchmakingQueueSize, size)
}

func (m *MetricRegistry) IncMatchmakingJoin() {
	atomic.AddInt64(&m.matchmakingJoins, 1)
}

func (m *MetricRegistry) IncMatchmakingLeave() {
	atomic.AddInt64(&m.matchmakingLeaves, 1)
}

func (m *MetricRegistry) IncMatchmakingMatch() {
	atomic.AddInt64(&m.matchmakingMatches, 1)
}

func (m *MetricRegistry) IncStockfishEvaluation() {
	atomic.AddInt64(&m.stockfishEvaluations, 1)
}

func normalizePath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if len(part) >= 20 || (len(part) > 6 && strings.ContainsAny(part, "0123456789-")) {
			parts[i] = "{id}"
		}
	}
	return "/" + strings.Join(parts, "/")
}

// PrometheusHandler outputs metrics in standard Prometheus exposition format (text/plain)
func (m *MetricRegistry) PrometheusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		uptimeSec := time.Since(m.startTime).Seconds()
		activeWS := atomic.LoadInt64(&m.activeWebSockets)
		if activeWS < 0 {
			activeWS = 0
		}
		activeGames := atomic.LoadInt64(&m.activeGames)
		if activeGames < 0 {
			activeGames = 0
		}
		queueSize := atomic.LoadInt64(&m.matchmakingQueueSize)
		if queueSize < 0 {
			queueSize = 0
		}
		evals := atomic.LoadInt64(&m.stockfishEvaluations)

		var sb strings.Builder

		// System / Go Runtime Metrics
		sb.WriteString("# HELP process_uptime_seconds Total seconds elapsed since server instance started\n")
		sb.WriteString("# TYPE process_uptime_seconds gauge\n")
		sb.WriteString(fmt.Sprintf("process_uptime_seconds{instance=\"%s\"} %.2f\n\n", m.instanceID, uptimeSec))

		sb.WriteString("# HELP go_goroutines Number of active goroutines\n")
		sb.WriteString("# TYPE go_goroutines gauge\n")
		sb.WriteString(fmt.Sprintf("go_goroutines{instance=\"%s\"} %d\n\n", m.instanceID, runtime.NumGoroutine()))

		sb.WriteString("# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use\n")
		sb.WriteString("# TYPE go_memstats_alloc_bytes gauge\n")
		sb.WriteString(fmt.Sprintf("go_memstats_alloc_bytes{instance=\"%s\"} %d\n\n", m.instanceID, mem.Alloc))

		sb.WriteString("# HELP go_memstats_sys_bytes Total bytes of memory obtained from OS\n")
		sb.WriteString("# TYPE go_memstats_sys_bytes gauge\n")
		sb.WriteString(fmt.Sprintf("go_memstats_sys_bytes{instance=\"%s\"} %d\n\n", m.instanceID, mem.Sys))

		sb.WriteString("# HELP go_memstats_num_gc Cumulative number of completed GC cycles\n")
		sb.WriteString("# TYPE go_memstats_num_gc counter\n")
		sb.WriteString(fmt.Sprintf("go_memstats_num_gc{instance=\"%s\"} %d\n\n", m.instanceID, mem.NumGC))

		// Application Metrics
		sb.WriteString("# HELP chesspasant_active_websocket_connections Current active WebSocket client connections\n")
		sb.WriteString("# TYPE chesspasant_active_websocket_connections gauge\n")
		sb.WriteString(fmt.Sprintf("chesspasant_active_websocket_connections{instance=\"%s\"} %d\n\n", m.instanceID, activeWS))

		sb.WriteString("# HELP chesspasant_active_games Current ongoing chess games on this instance\n")
		sb.WriteString("# TYPE chesspasant_active_games gauge\n")
		sb.WriteString(fmt.Sprintf("chesspasant_active_games{instance=\"%s\"} %d\n\n", m.instanceID, activeGames))

		sb.WriteString("# HELP chesspasant_matchmaking_queue_size Number of players waiting in matchmaking queue\n")
		sb.WriteString("# TYPE chesspasant_matchmaking_queue_size gauge\n")
		sb.WriteString(fmt.Sprintf("chesspasant_matchmaking_queue_size{instance=\"%s\"} %d\n\n", m.instanceID, queueSize))

		sb.WriteString("# HELP chesspasant_matchmaking_joins_total Total players who joined matchmaking queue\n")
		sb.WriteString("# TYPE chesspasant_matchmaking_joins_total counter\n")
		sb.WriteString(fmt.Sprintf("chesspasant_matchmaking_joins_total{instance=\"%s\"} %d\n\n", m.instanceID, atomic.LoadInt64(&m.matchmakingJoins)))

		sb.WriteString("# HELP chesspasant_matchmaking_leaves_total Total players who left matchmaking queue\n")
		sb.WriteString("# TYPE chesspasant_matchmaking_leaves_total counter\n")
		sb.WriteString(fmt.Sprintf("chesspasant_matchmaking_leaves_total{instance=\"%s\"} %d\n\n", m.instanceID, atomic.LoadInt64(&m.matchmakingLeaves)))

		sb.WriteString("# HELP chesspasant_matchmaking_matches_total Total matches formed by matchmaking\n")
		sb.WriteString("# TYPE chesspasant_matchmaking_matches_total counter\n")
		sb.WriteString(fmt.Sprintf("chesspasant_matchmaking_matches_total{instance=\"%s\"} %d\n\n", m.instanceID, atomic.LoadInt64(&m.matchmakingMatches)))

		sb.WriteString("# HELP chesspasant_stockfish_evaluations_total Total engine analysis requests processed\n")
		sb.WriteString("# TYPE chesspasant_stockfish_evaluations_total counter\n")
		sb.WriteString(fmt.Sprintf("chesspasant_stockfish_evaluations_total{instance=\"%s\"} %d\n\n", m.instanceID, evals))

		// HTTP Requests Total
		sb.WriteString("# HELP http_requests_total Total number of HTTP requests by method, path, and status code\n")
		sb.WriteString("# TYPE http_requests_total counter\n")

		m.httpRequestsMu.RLock()
		reqKeys := make([]string, 0, len(m.httpRequestsTotal))
		for k := range m.httpRequestsTotal {
			reqKeys = append(reqKeys, k)
		}
		sort.Strings(reqKeys)
		for _, k := range reqKeys {
			parts := strings.Split(k, "|")
			if len(parts) == 3 {
				cnt := atomic.LoadInt64(m.httpRequestsTotal[k])
				sb.WriteString(fmt.Sprintf("http_requests_total{instance=\"%s\",method=\"%s\",path=\"%s\",status=\"%s\"} %d\n",
					m.instanceID, parts[0], parts[1], parts[2], cnt))
			}
		}
		m.httpRequestsMu.RUnlock()
		sb.WriteString("\n")

		// HTTP Request Duration Summary
		sb.WriteString("# HELP http_request_duration_ms_avg Average request processing latency in milliseconds\n")
		sb.WriteString("# TYPE http_request_duration_ms_avg gauge\n")

		m.httpDurationMu.RLock()
		durKeys := make([]string, 0, len(m.httpDurationMsSum))
		for k := range m.httpDurationMsSum {
			durKeys = append(durKeys, k)
		}
		sort.Strings(durKeys)
		for _, k := range durKeys {
			parts := strings.Split(k, "|")
			if len(parts) == 2 {
				sum := atomic.LoadInt64(m.httpDurationMsSum[k])
				cnt := atomic.LoadInt64(m.httpDurationMsCount[k])
				var avg float64
				if cnt > 0 {
					avg = float64(sum) / float64(cnt)
				}
				sb.WriteString(fmt.Sprintf("http_request_duration_ms_avg{instance=\"%s\",method=\"%s\",path=\"%s\"} %.2f\n",
					m.instanceID, parts[0], parts[1], avg))
			}
		}
		m.httpDurationMu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sb.String()))
	}
}

// JSONHandler returns human/API friendly JSON format for health dashboards
func (m *MetricRegistry) JSONHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		m.httpRequestsMu.RLock()
		reqStats := make(map[string]int64, len(m.httpRequestsTotal))
		for k, v := range m.httpRequestsTotal {
			reqStats[k] = atomic.LoadInt64(v)
		}
		m.httpRequestsMu.RUnlock()

		m.httpDurationMu.RLock()
		latencyStats := make(map[string]map[string]interface{}, len(m.httpDurationMsSum))
		for k := range m.httpDurationMsSum {
			sum := atomic.LoadInt64(m.httpDurationMsSum[k])
			cnt := atomic.LoadInt64(m.httpDurationMsCount[k])
			var avg float64
			if cnt > 0 {
				avg = float64(sum) / float64(cnt)
			}
			latencyStats[k] = map[string]interface{}{
				"total_ms": sum,
				"count":    cnt,
				"avg_ms":   avg,
			}
		}
		m.httpDurationMu.RUnlock()

		payload := map[string]interface{}{
			"instanceId": m.instanceID,
			"uptimeSec":  time.Since(m.startTime).Seconds(),
			"runtime": map[string]interface{}{
				"goroutines":   runtime.NumGoroutine(),
				"allocBytes":   mem.Alloc,
				"totalAlloc":   mem.TotalAlloc,
				"sysBytes":     mem.Sys,
				"numGC":        mem.NumGC,
				"heapObjects":  mem.HeapObjects,
			},
			"chess": map[string]interface{}{
				"activeWebSockets":      atomic.LoadInt64(&m.activeWebSockets),
				"activeGames":           atomic.LoadInt64(&m.activeGames),
				"matchmakingQueueSize":  atomic.LoadInt64(&m.matchmakingQueueSize),
				"matchmakingJoins":      atomic.LoadInt64(&m.matchmakingJoins),
				"matchmakingLeaves":     atomic.LoadInt64(&m.matchmakingLeaves),
				"matchmakingMatches":    atomic.LoadInt64(&m.matchmakingMatches),
				"stockfishEvaluations":  atomic.LoadInt64(&m.stockfishEvaluations),
			},
			"http": map[string]interface{}{
				"requests": reqStats,
				"latencies": latencyStats,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload)
	}
}
