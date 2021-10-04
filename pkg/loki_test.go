package pkg

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	speedyTesting "speedy/pkg/testing"
	"testing"
)
import "github.com/stretchr/testify/assert"

// Test_NewLokiStreams ensures that NewLokiStreams works as expected.
func Test_NewLokiStreams(t *testing.T) {
	streams := NewLokiStreams(1000, math.MaxInt32)
	assert.NotNil(t, streams)
	assert.Equal(t, 1000, cap(streams.Streams))
	assert.Equal(t, 0, len(streams.Streams))
	assert.Equal(t, math.MaxInt32, streams.bufferMaxByteSize)
	assert.Equal(t, 1000, streams.bufferMaxBatchSize)
}

func Test_LokiClientFactoryCreate(t *testing.T) {
	var tests = []struct {
		clientName  string
		expectedNil bool
	}{
		{
			"proto",
			false,
		},
		{
			"http",
			false,
		},
		{
			"batman",
			true,
		},
	}
	for index, tt := range tests {
		t.Run(fmt.Sprintf("test_%d", index), func(t *testing.T) {
			lokiClient := LokiClientFactoryCreate(tt.clientName, "https://loki.com/loki/api/v1/push")
			if tt.expectedNil {
				assert.Nil(t, lokiClient)
			} else {
				assert.NotNil(t, lokiClient)
			}
		})

	}

}

// Test_NewLokiHttpClient_SendData ensures that SendData from LokiHttpClient works as expected.
func Test_NewLokiHttpClient_SendData(t *testing.T) {
	var lastRequest *http.Request = nil

	client := &LokiHttpClient{lokiUrl: "https://loki.com/loki/api/v1/push", HttpClient: &http.Client{}}
	client.SetHttpClient(speedyTesting.NewTestClient(func(req *http.Request) *http.Response {
		lastRequest = req
		return &http.Response{
			StatusCode: 204,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString("")),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	}))
	assert.NotNil(t, client)

	dummyData := LokiStreams{
		Streams: []LokiStream{{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"0", "log-line"}},
		}},
		Count: 0,
	}
	err := client.SendData(context.Background(), &dummyData)
	assert.Nil(t, err)

	requestBody, err := ioutil.ReadAll(lastRequest.Body)
	assert.Nil(t, err)

	assert.Equal(t, "{\"streams\":[{\"stream\":{\"label1\":\"value\"},\"values\":[[\"0\",\"log-line\"]]}]}", string(requestBody))
}

// Test_NewLokiProtoClient_SendData ensures that SendData from LokiHttpClient works as expected.
func Test_NewLokiProtoClient_SendData(t *testing.T) {
	var lastRequest *http.Request = nil

	client := &LokiHttpClient{lokiUrl: "https://loki.com/loki/api/v1/push", HttpClient: &http.Client{}}
	client.SetHttpClient(speedyTesting.NewTestClient(func(req *http.Request) *http.Response {
		lastRequest = req
		return &http.Response{
			StatusCode: 204,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString("")),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	}))
	assert.NotNil(t, client)

	dummyData := LokiStreams{
		Streams: []LokiStream{{
			Labels: map[string]string{
				"label1": "value",
			},
			Values: [][]string{{"0", "log-line"}},
		}},
		Count: 0,
	}
	err := client.SendData(context.Background(), &dummyData)
	assert.Nil(t, err)

	requestBody, err := ioutil.ReadAll(lastRequest.Body)
	assert.Nil(t, err)
	assert.NotEmpty(t, requestBody)
}
