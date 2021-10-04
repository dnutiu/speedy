package pkg

import (
	"bytes"
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/goccy/go-json"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"io"
	"io/ioutil"
	"net/http"
	"speedy/pkg/logproto"
	"strings"
	"time"
)

// LokiClientFactoryCreate is a factory for creating Loki clients.
func LokiClientFactoryCreate(clientName string, lokiUrl string) ISpeedySink {
	if clientName == "http" {
		return NewLokiHttpClient(lokiUrl)
	} else if clientName == "proto" {
		return NewLokiProtoClient(lokiUrl)
	}
	return nil
}

// LokiHttpClient is a simple ISpeedySink that sends data to Loki via HTTP protocol.
type LokiHttpClient struct {
	lokiUrl    string
	HttpClient *http.Client
}

// NewLokiHttpClient constructs a new instance of LokiHttpClient.
func NewLokiHttpClient(lokiUrl string) ISpeedySink {
	return &LokiHttpClient{lokiUrl: lokiUrl, HttpClient: &http.Client{}}
}

// SendData sends LokiStreams to Loki over HTTP with JSON packaging.
func (l *LokiHttpClient) SendData(ctx context.Context, data *LokiStreams) error {
	b, err := json.Marshal(data)
	if err != nil {
		SugaredLogger.Error(err)
		sentry.CaptureException(err)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", l.lokiUrl, bytes.NewBuffer(b))
	if err != nil {
		SugaredLogger.Errorf("failed to create new POST request: %s", err)
		sentry.CaptureException(err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.HttpClient.Do(req)
	if err != nil {
		SugaredLogger.Error("failed to execute request: %s", err)
		sentry.CaptureException(err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			sentry.CaptureException(err)
			SugaredLogger.Error(err)
		}
	}(resp.Body)

	if resp.StatusCode != 204 {
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			SugaredLogger.Debugf("{%d} - {%s}", resp.StatusCode, string(responseBody))
		} else {
			SugaredLogger.Debugf("%v", err)
		}
		sentry.CaptureException(err)
	} else {
		SugaredLogger.Debugf("{%d}", resp.StatusCode)
	}

	return nil
}

// SetHttpClient replaces the default http speedySink.
func (l *LokiHttpClient) SetHttpClient(client *http.Client) {
	l.HttpClient = client
}

// Shutdown shuts down the idle connections
func (l *LokiHttpClient) Shutdown() {
	l.HttpClient.CloseIdleConnections()
}

// LokiProtoClient is a simple ISpeedySink that sends data to Loki via snappy-compressed protocol buffers protocol.
type LokiProtoClient struct {
	lokiUrl    string
	HttpClient *http.Client
}

// NewLokiProtoClient constructs a new instance of LokiProtoClient.
func NewLokiProtoClient(lokiUrl string) ISpeedySink {
	return &LokiProtoClient{lokiUrl: lokiUrl, HttpClient: &http.Client{}}
}

// SendData sends LokiStreams to Loki over protocol buffers.
func (l *LokiProtoClient) SendData(ctx context.Context, data *LokiStreams) error {
	pushRequest := logproto.PushRequest{
		Streams: make([]logproto.Stream, 0, data.Count),
	}

	// Format labels and append data to stream.
	for _, entry := range data.Streams {
		tmpLabels := make([]string, 0, 2)
		for k, v := range entry.Labels {
			tmpLabels = append(tmpLabels, fmt.Sprintf("%s=%q", k, v))
		}
		labels := fmt.Sprintf("{%s}", strings.Join(tmpLabels, ", "))

		pushRequest.Streams = append(pushRequest.Streams, logproto.Stream{
			Labels: labels,
			Entries: []logproto.Entry{{
				Timestamp: time.Now().UTC(),
				Line:      entry.Values[0][1],
			}},
		})
	}

	// Marshall into protobuf and snappy encode.
	buf, err := proto.Marshal(&pushRequest)
	if err != nil {
		SugaredLogger.Errorf("Failed to marshall into protobuffer %s", err)
		return err
	}
	b := snappy.Encode(nil, buf)

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", l.lokiUrl, bytes.NewBuffer(b))
	if err != nil {
		SugaredLogger.Errorf("failed to create new POST request: %s", err)
		sentry.CaptureException(err)
		return err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := l.HttpClient.Do(req)
	if err != nil {
		SugaredLogger.Error("failed to execute request: %s", err)
		sentry.CaptureException(err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			SugaredLogger.Error(err)
			sentry.CaptureException(err)
		}
	}(resp.Body)

	if resp.StatusCode != 204 {
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			SugaredLogger.Debugf("{%d} - {%s}", resp.StatusCode, string(responseBody))
		} else {
			SugaredLogger.Debugf("%v", err)
		}
		sentry.CaptureException(err)
	} else {
		SugaredLogger.Debugf("{%d}", resp.StatusCode)
	}

	return nil
}

// SetHttpClient replaces the default http speedySink.
func (l *LokiProtoClient) SetHttpClient(client *http.Client) {
	l.HttpClient = client
}

// Shutdown shuts down the idle connections
func (l *LokiProtoClient) Shutdown() {
	l.HttpClient.CloseIdleConnections()
}
