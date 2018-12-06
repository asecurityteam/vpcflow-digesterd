package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type payload struct {
	ID    string `json:"id"`
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

// DigestQueuer is a Queuer implementation which queues digest jobs onto a streaming appliance
type DigestQueuer struct {
	Endpoint *url.URL
	Topic    string
	Client   *http.Client
}

// Queue enqueues a digest job onto a streaming appliance
func (q *DigestQueuer) Queue(ctx context.Context, id string, start, stop time.Time) error {
	body := payload{
		ID:    id,
		Start: start.Format(time.RFC3339Nano),
		Stop:  stop.Format(time.RFC3339Nano),
	}
	rawBody, _ := json.Marshal(body)
	path, _ := url.Parse(fmt.Sprintf("%s/event", q.Topic))
	u := q.Endpoint.ResolveReference(path)
	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(rawBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := q.Client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response from streaming appliance: %d", res.StatusCode)
	}
	return nil
}
