package pkg

import (
	"context"
	"github.com/stretchr/testify/assert"
	"math"
	speedyTesting "speedy/pkg/testing"
	"testing"
	"time"
)

// SpeedyTestSink is a sink used for internal testing.
type SpeedyTestSink struct {
	savedData       *LokiStreams
	sendDataCounter int
	shutdown        bool
}

func (p *SpeedyTestSink) SendData(_ context.Context, data *LokiStreams) error {
	p.sendDataCounter += 1
	p.savedData = data
	return nil
}

func (p *SpeedyTestSink) Shutdown() {
	p.shutdown = true
}

// Test_Pusher_RunForever_Batch ensure that the pushes batches items correctly.
func Test_Pusher_RunForever_Batch(t *testing.T) {
	client := &SpeedyTestSink{}
	lokiPusher := NewPusher(client, 3, math.MaxInt32)
	lokiPusher.TimeProvider = speedyTesting.ZeroNanoTimeProvider
	go lokiPusher.RunForever()

	data := []LokiStream{
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"0", "log-line-0"}},
		},
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"1", "log-line-1"}},
		},
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"2", "log-line-2"}},
		},
	}

	lokiPusher.DataChannel <- data[0]
	lokiPusher.DataChannel <- data[1]
	lokiPusher.DataChannel <- data[2]
	time.Sleep(100 * time.Millisecond)
	lokiPusher.Shutdown()

	assert.Equal(t, data, client.savedData.Streams)
}

// Test_Pusher_RunForever_Ticker ensure that the pusher flushes on stale batches and on shutdown.
func Test_Pusher_RunForever_Ticker(t *testing.T) {
	client := &SpeedyTestSink{}
	lokiPusher := NewPusher(client, 3, math.MaxInt32)
	lokiPusher.TimeProvider = speedyTesting.ZeroNanoTimeProvider
	lokiPusher.SecondsToFlush = 50 * time.Millisecond
	go lokiPusher.RunForever()

	first := []LokiStream{{
		Labels: map[string]string{
			"label1": "value",
		},
		Values: [][]string{{"0", "log-line-0"}},
	},
	}

	lokiPusher.DataChannel <- first[0]

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, first, client.savedData.Streams)

	second := []LokiStream{
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"1", "log-line-1"}},
		},
	}

	lokiPusher.DataChannel <- second[0]

	time.Sleep(100 * time.Millisecond)
	lokiPusher.Shutdown()

	assert.Equal(t, second, client.savedData.Streams)
}

// Test_Pusher_RunForever_BatchBytes ensure that it pushes batches items correctly according to their size.
func Test_Pusher_RunForever_BatchBytes(t *testing.T) {
	sink := &SpeedyTestSink{}
	lokiPusher := NewPusher(sink, 3, 16)
	lokiPusher.TimeProvider = speedyTesting.ZeroNanoTimeProvider
	go lokiPusher.RunForever()

	data := []LokiStream{
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"0", "log-line-0"}},
			Size:   10,
		},
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"1", "log-line-1"}},
			Size:   10,
		},
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"2", "log-line-2"}},
			Size:   10,
		},
	}
	lokiPusher.DataChannel <- data[0]
	lokiPusher.DataChannel <- data[1]
	lokiPusher.DataChannel <- data[2]

	time.Sleep(100 * time.Millisecond)
	lokiPusher.Shutdown()

	assert.Equal(t, []LokiStream{data[0], data[1]}, sink.savedData.Streams)
}

// Test_Pusher_RunForever_Shutdown ensures a clean shutdown and flush regardless of batch size.
func Test_Pusher_RunForever_Shutdown(t *testing.T) {
	sink := &SpeedyTestSink{}

	lokiPusher := NewPusher(sink, 3, 16)
	lokiPusher.TimeProvider = speedyTesting.ZeroNanoTimeProvider
	go lokiPusher.RunForever()

	data := []LokiStream{
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"0", "log-line-0"}},
			Size:   10,
		},
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"1", "log-line-1"}},
			Size:   10,
		},
		{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"2", "log-line-2"}},
			Size:   10,
		},
	}

	lokiPusher.DataChannel <- data[0]
	lokiPusher.DataChannel <- data[1]
	lokiPusher.DataChannel <- data[2]

	lokiPusher.Shutdown()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 2, sink.sendDataCounter)
	assert.Equal(t, []LokiStream{data[2]}, sink.savedData.Streams)
}
