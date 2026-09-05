// Package gatewaymonitor keeps an eye on registered gateway APIs: it probes
// each upstream and tails the per-API nginx access logs (JSONL on the shared
// registry volume) to feed metrics into Redis + an in-memory recent buffer.
package gatewaymonitor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"go-enterprise-api/internal/domain/gatewayapi"
)

// ── Health checks ──────────────────────────────────────────────────────────

type HealthChecker struct {
	Repo     gatewayapi.Repository
	Client   *http.Client
	Interval time.Duration
}

func NewHealthChecker(repo gatewayapi.Repository, interval time.Duration, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		Repo:     repo,
		Client:   &http.Client{Timeout: timeout},
		Interval: interval,
	}
}

// Run probes every active upstream until ctx is cancelled.
func (hc *HealthChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(hc.Interval)
	defer ticker.Stop()
	hc.checkOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkOnce(ctx)
		}
	}
}

func (hc *HealthChecker) checkOnce(ctx context.Context) {
	apis, err := hc.Repo.ListActive(ctx)
	if err != nil {
		return
	}
	checkedAt := time.Now().UTC()
	for _, api := range apis {
		endpoint, err := healthEndpoint(api.Upstream)
		if err != nil {
			continue
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if req == nil {
			continue
		}
		status := "unhealthy"
		if resp, err := hc.Client.Do(req); err == nil {
			// Any HTTP response counts as reachable; only transport errors are
			// treated as unhealthy (404 on a non-/healthz upstream still means
			// the route is alive).
			_ = resp.Body.Close()
			status = "healthy"
		}
		_ = hc.Repo.UpdateStatus(ctx, api.ID, status, checkedAt)
	}
}

func healthEndpoint(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", gatewayapi.ErrInvalidUpstream
	}
	u.Path = "/healthz"
	u.RawQuery = ""
	return u.String(), nil
}

// ── Stats aggregation (from nginx JSON access logs) ────────────────────────

type Entry struct {
	APIID    int64   `json:"-"`
	Time     string  `json:"time"`
	IP       string  `json:"ip"`
	Status   int     `json:"status"`
	Method   string  `json:"method"`
	URI      string  `json:"uri"`
	Bytes    int     `json:"bytes"`
	Upstream string  `json:"upstream"`
	RT       float64 `json:"rt"`
	UA       string  `json:"ua"`
}

type TodayStats struct {
	Count      int64 `json:"count"`
	H2xx       int64 `json:"2xx"`
	H4xx       int64 `json:"4xx"`
	H5xx       int64 `json:"5xx"`
	LatencyMS  int64 `json:"latency_ms"`
	ErrorCount int64 `json:"errors"`
}

// Aggregator tails the JSONL access logs written by nginx for each registered
// API, accumulates today's totals in Redis and keeps a small ring buffer of
// recent entries in memory.
type Aggregator struct {
	RDB     *redis.Client
	LogsDir string
	Recent  map[int64][]Entry
	mu      sync.Mutex
	offsets map[string]int64
	recentN int
}

func NewAggregator(rdb *redis.Client, logsDir string, recentN int) *Aggregator {
	if recentN <= 0 {
		recentN = 100
	}
	return &Aggregator{
		RDB:     rdb,
		LogsDir: logsDir,
		Recent:  make(map[int64][]Entry),
		offsets: make(map[string]int64),
		recentN: recentN,
	}
}

// Run tails the log directory every 2s until ctx is cancelled.
func (a *Aggregator) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.Tail()
		}
	}
}

// Tail reads any new lines from the per-API log files and records them.
func (a *Aggregator) Tail() {
	entries, err := os.ReadDir(a.LogsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var id int64
		if _, err := fmt.Sscanf(e.Name(), "reg_%d.json", &id); err != nil {
			continue
		}
		a.tailFile(filepath.Join(a.LogsDir, e.Name()), id)
	}
}

func (a *Aggregator) tailFile(path string, apiID int64) {
	a.mu.Lock()
	offset := a.offsets[path]
	f, err := os.Open(path)
	if err != nil {
		a.mu.Unlock()
		return
	}
	info, err := f.Stat()
	if err == nil && info.Size() < offset {
		offset = 0 // rotated
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		a.mu.Unlock()
		_ = f.Close()
		return
	}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 256*1024)
	newOffset := offset
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		newOffset += int64(len(scanner.Bytes()) + 1)
		if line == "" {
			continue
		}
		var e Entry
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		e.APIID = apiID
		a.record(e)
	}
	a.offsets[path] = newOffset
	a.mu.Unlock()
	_ = f.Close()
}

// record appends one log line to Redis counters and the recent ring. The
// caller (tailFile) must hold a.mu for the ring update.
func (a *Aggregator) record(e Entry) {
	now := time.Now().UTC()
	day := now.Format("20060102")
	key := "gwstats:" + strconv.FormatInt(e.APIID, 10) + ":" + day
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pipe := a.RDB.Pipeline()
	pipe.HIncrBy(ctx, key, "count", 1)
	pipe.HIncrBy(ctx, key, "rt_total_ms", int64(e.RT*1000))
	switch {
	case e.Status >= 500:
		pipe.HIncrBy(ctx, key, "5xx", 1)
		pipe.HIncrBy(ctx, key, "errors", 1)
	case e.Status >= 400:
		pipe.HIncrBy(ctx, key, "4xx", 1)
		pipe.HIncrBy(ctx, key, "errors", 1)
	case e.Status >= 200:
		pipe.HIncrBy(ctx, key, "2xx", 1)
	default:
		pipe.HIncrBy(ctx, key, "errors", 1)
	}
	pipe.Expire(ctx, key, 48*time.Hour)
	_, _ = pipe.Exec(ctx)

	ring := a.Recent[e.APIID]
	if len(ring) >= a.recentN {
		ring = append(ring[:copy(ring, ring[1:])], e)
	} else {
		ring = append(ring, e)
	}
	a.Recent[e.APIID] = ring
}

// Stats aggregates today's totals from Redis plus recent entries.
func (a *Aggregator) Stats(ctx context.Context, apiID int64) (TodayStats, []Entry) {
	day := time.Now().UTC().Format("20060102")
	key := "gwstats:" + strconv.FormatInt(apiID, 10) + ":" + day

	var ts TodayStats
	vals, err := a.RDB.HGetAll(ctx, key).Result()
	if err == nil {
		n, _ := strconv.ParseInt(vals["count"], 10, 64)
		ts.Count = n
		ts.H2xx, _ = strconv.ParseInt(vals["2xx"], 10, 64)
		ts.H4xx, _ = strconv.ParseInt(vals["4xx"], 10, 64)
		ts.H5xx, _ = strconv.ParseInt(vals["5xx"], 10, 64)
		ts.LatencyMS, _ = strconv.ParseInt(vals["rt_total_ms"], 10, 64)
		ts.ErrorCount, _ = strconv.ParseInt(vals["errors"], 10, 64)
	}
	if ts.Count > 0 {
		ts.LatencyMS = ts.LatencyMS / ts.Count
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	recent := make([]Entry, 0, len(a.Recent[apiID]))
	recent = append(recent, a.Recent[apiID]...)
	return ts, recent
}
