package pkg

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// ISpeedySink is the interface for Speedy sinks.
type ISpeedySink interface {
	SendData(context context.Context, data *LokiStreams) error
	Shutdown()
}

// LokiStream represents stream of data that Loki push API accepts.
type LokiStream struct {
	// Labels is the key-value map used as the Labels in Loki.
	Labels map[string]string `json:"stream"`
	// Values an array of values.
	Values [][]string `json:"values"`
	// Size is the size of the current struct in bytes.
	Size int `json:"-"`
}

// LokiStreams represents a list of LokiStream that Loki push API accepts.
type LokiStreams struct {
	// Streams is an array of LokiStream.
	Streams []LokiStream `json:"streams"`
	// Count represents the number of LokiStream's in the struct.
	Count int `json:"-"`
	// TotalSize is the total size if the LokiStreams from struct.
	TotalSize          int `json:"-"`
	bufferMaxBatchSize int
	bufferMaxByteSize  int
}

func (ls *LokiStreams) AddData(s LokiStream) {
	ls.Count += 1
	ls.Streams = append(ls.Streams, s)
	ls.TotalSize += s.Size
}

// IsFull returns whether the LokiStreams is full by checking against buffer_max_bytes and then buffer_max_batch_size.
// If buffer_max_bytes is 0 then IsFull only checks against buffer_max_batch_size.
func (ls *LokiStreams) IsFull() bool {
	if ls.TotalSize >= ls.bufferMaxByteSize {
		return true
	}
	if ls.Count >= ls.bufferMaxBatchSize {
		return true
	}
	return false
}

// NewLokiStreams creates a new instance of LokiStreams of max capacity
func NewLokiStreams(maxCapacity int, maxSizeBytes int) *LokiStreams {
	return &LokiStreams{
		Streams:            make([]LokiStream, 0, maxCapacity),
		Count:              0,
		bufferMaxBatchSize: maxCapacity,
		bufferMaxByteSize:  maxSizeBytes,
	}
}

// Pusher ensures that messages are efficiently pushed into Sinks.
type Pusher struct {
	// DataChannel is a LokiStream channel that is used to send data to the pusher.
	DataChannel       chan LokiStream
	TimeProvider      func() string
	speedySink        ISpeedySink
	lastFlush         time.Time
	SecondsToFlush    time.Duration
	currentStreams    *LokiStreams
	maxBatchSize      int
	maxBatchSizeBytes int
	shutdownChannel   chan int
}

// UnixNanoTimeProvider provides time as a string in unix nanoseconds.
func UnixNanoTimeProvider() string {
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}

// NewPusher creates a new Loki pusher instance.
func NewPusher(sink ISpeedySink, maxBatchSize int, maxBatchSizeBytes int) *Pusher {
	if sink == nil {
		panic("Speedy sink is nil")
	}

	return &Pusher{
		DataChannel:       make(chan LokiStream, 1000),
		TimeProvider:      UnixNanoTimeProvider,
		SecondsToFlush:    1 * time.Minute,
		speedySink:        sink,
		lastFlush:         time.Now(),
		maxBatchSize:      maxBatchSize,
		maxBatchSizeBytes: maxBatchSizeBytes,
		currentStreams:    NewLokiStreams(maxBatchSize, maxBatchSizeBytes),
		shutdownChannel:   make(chan int),
	}
}

// RunForever runs the pusher forever, or until Shutdown is called.
func (lp *Pusher) RunForever() {
	var mutex = &sync.Mutex{}
	tick := time.Tick(lp.SecondsToFlush)

	for {
		select {
		case data := <-lp.DataChannel:
			mutex.Lock()
			// This is sort of bad but Loki does not support out of order messages, since have N goroutines
			// we will have to override the timestamp here, otherwise the timestamps may clash or be out of order.
			data.Values[0][0] = lp.TimeProvider()
			lp.currentStreams.AddData(data)
			if lp.currentStreams.IsFull() {
				lp.flushCurrentBatch()
			}
			mutex.Unlock()
		case <-lp.shutdownChannel:
			// Ensure clean shutdown.
			mutex.Lock()
			//goland:noinspection ALL
			defer mutex.Unlock()
			SugaredLogger.Info("Shutting down Pusher. Draining")
			for {
				isDone := false
				select {
				case data := <-lp.DataChannel:
					data.Values[0][0] = lp.TimeProvider()
					lp.currentStreams.AddData(data)
					if lp.currentStreams.IsFull() {
						lp.flushCurrentBatch()
					}
				default:
					isDone = true
				}
				if isDone {
					break
				}
			}
			SugaredLogger.Info("Drained.")
			lp.flushCurrentBatch()
			lp.speedySink.Shutdown()
			return
		case <-tick:
			// This branch will handle periodical flushes so that the pipeline won't remain stale.
			mutex.Lock()
			if time.Now().Sub(lp.lastFlush).Milliseconds() >= lp.SecondsToFlush.Milliseconds() {
				lp.flushCurrentBatch()
			}
			mutex.Unlock()
		}
	}
}

// flushCurrentBatch flushes the current batch.
func (lp *Pusher) flushCurrentBatch() {
	// Skip flushing, no data.
	if lp.currentStreams.Count == 0 {
		return
	}
	err := lp.speedySink.SendData(context.Background(), lp.currentStreams)
	if err != nil {
		SugaredLogger.Error(err)
	}
	lp.lastFlush = time.Now()
	lp.currentStreams = NewLokiStreams(lp.maxBatchSize, lp.maxBatchSizeBytes)
}

// Shutdown shutdowns the Loki pusher.
func (lp *Pusher) Shutdown() {
	lp.shutdownChannel <- 1
}
