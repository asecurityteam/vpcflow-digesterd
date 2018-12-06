package stream

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	baseURL = "http://some.host"
	topic   = "my-topic"
)

// testRoundTripper is a stub for testing requests made by an HTTP client
type testRoundTripper struct {
	StatusCode  int
	Error       error
	ExpectedURL string
}

func (rt *testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if rt.ExpectedURL != r.URL.String() {
		return nil, fmt.Errorf("Expected %s, but was %s", rt.ExpectedURL, r.URL.String())
	}
	if r.Header.Get("Content-Type") != "application/octet-stream" {
		return nil, fmt.Errorf("Expected Content-Type to be set to application/octet-stream, but was %s", r.Header.Get("Content-Type"))
	}
	if rt.Error != nil {
		return nil, rt.Error
	}
	res := &http.Response{
		StatusCode: rt.StatusCode,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
	}
	return res, nil
}

func TestDigestQueuer(t *testing.T) {
	endpoint, _ := url.Parse(baseURL)
	expectedURL := fmt.Sprintf("%s/%s/event", baseURL, topic)
	client := &http.Client{
		Transport: &testRoundTripper{
			StatusCode:  200,
			ExpectedURL: expectedURL,
		},
	}
	dq := DigestQueuer{
		Client:   client,
		Endpoint: endpoint,
		Topic:    topic,
	}
	err := dq.Queue(context.Background(), "digestId", time.Now(), time.Now())
	assert.Nil(t, err)
}

func TestUnexpectedResponse(t *testing.T) {
	endpoint, _ := url.Parse(baseURL)
	expectedURL := fmt.Sprintf("%s/%s/event", baseURL, topic)
	client := &http.Client{
		Transport: &testRoundTripper{
			StatusCode:  500,
			ExpectedURL: expectedURL,
		},
	}
	dq := DigestQueuer{
		Client:   client,
		Endpoint: endpoint,
		Topic:    topic,
	}
	err := dq.Queue(context.Background(), "digestId", time.Now(), time.Now())
	assert.NotNil(t, err)
}

func TestRTError(t *testing.T) {
	endpoint, _ := url.Parse(baseURL)
	expectedURL := fmt.Sprintf("%s/%s/event", baseURL, topic)
	client := &http.Client{
		Transport: &testRoundTripper{
			Error:       errors.New("oops"),
			StatusCode:  200,
			ExpectedURL: expectedURL,
		},
	}
	dq := DigestQueuer{
		Client:   client,
		Endpoint: endpoint,
		Topic:    topic,
	}
	err := dq.Queue(context.Background(), "digestId", time.Now(), time.Now())
	assert.NotNil(t, err)
}
