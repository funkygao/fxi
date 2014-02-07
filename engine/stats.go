package engine

import (
	"fmt"
	"github.com/funkygao/golib/gofmt"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

type AtomicInt int64

func (this *AtomicInt) Add(n int64) {
	atomic.AddInt64((*int64)(this), n)
}

func (this *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(this))
}

func (this *AtomicInt) String() string {
	return strconv.FormatInt(this.Get(), 10)
}

type engineStats struct {
	startedAt time.Time
	MemStats  *runtime.MemStats

	TotalSessions    AtomicInt
	TotalCalls       AtomicInt
	TotalFailedCalls AtomicInt

	TotalRequests  map[string]AtomicInt // key is client ip
	PeriodRequests map[string]AtomicInt // key is client ip
}

func newEngineStats() (this *engineStats) {
	this = new(engineStats)
	this.MemStats = new(runtime.MemStats)
	this.TotalRequests = make(map[string]AtomicInt)
	this.PeriodRequests = make(map[string]AtomicInt)
	return
}

func (this engineStats) Start(t time.Time) {
	this.startedAt = t
}

func (this *engineStats) Runtime() map[string]interface{} {
	this.refreshMemStats()

	s := make(map[string]interface{})
	s["goroutines"] = runtime.NumGoroutine()
	s["memory.allocated"] = gofmt.ByteSize(this.MemStats.Alloc).String()
	s["memory.mallocs"] = gofmt.ByteSize(this.MemStats.Mallocs).String()
	s["memory.frees"] = gofmt.ByteSize(this.MemStats.Frees).String()
	s["memory.last_gc"] = this.MemStats.LastGC
	s["memory.gc.num"] = this.MemStats.NumGC
	s["memory.gc.num_per_second"] = float64(this.MemStats.NumGC) / time.
		Since(this.startedAt).Seconds()
	s["memory.gc.total_pause"] = fmt.Sprintf("%dms",
		this.MemStats.PauseTotalNs/uint64(time.Millisecond))
	s["memory.heap.alloc"] = gofmt.ByteSize(this.MemStats.HeapAlloc).String()
	s["memory.heap.sys"] = gofmt.ByteSize(this.MemStats.HeapSys).String()
	s["memory.heap.idle"] = gofmt.ByteSize(this.MemStats.HeapIdle).String()
	s["memory.heap.released"] = gofmt.ByteSize(this.MemStats.HeapReleased).String()
	s["memory.heap.objects"] = gofmt.Comma(int64(this.MemStats.HeapObjects))
	s["memory.stack"] = gofmt.ByteSize(this.MemStats.StackInuse).String()
	gcPausesMs := make([]string, 0, 20)
	for _, pauseNs := range this.MemStats.PauseNs {
		if pauseNs == 0 {
			continue
		}

		pauseStr := fmt.Sprintf("%dms",
			pauseNs/uint64(time.Millisecond))
		if pauseStr == "0ms" {
			continue
		}

		gcPausesMs = append(gcPausesMs, pauseStr)
	}
	s["memory.gc.pauses"] = gcPausesMs

	return s
}

func (this *engineStats) refreshMemStats() {
	runtime.ReadMemStats(this.MemStats)
}
