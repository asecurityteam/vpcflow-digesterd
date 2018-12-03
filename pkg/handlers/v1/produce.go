package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
)

type payload struct {
	ID    string `json:"id"`
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

// Produce is a handler which performs the digest job, and stores the digest
type Produce struct {
	LogProvider      types.LoggerProvider
	StatProvider     types.StatsProvider
	Storage          types.Storage
	Marker           types.Marker
	DigesterProvider types.DigesterProvider
}

// ServeHTTP handles incoming HTTP requests, and creates a vpc flow digest
func (h *Produce) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body payload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeTextResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if body.ID == "" {
		msg := "missing ID field"
		writeTextResponse(w, http.StatusBadRequest, msg)
		return
	}

	start, err := time.Parse(time.RFC3339Nano, body.Start)
	if err != nil {
		writeTextResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	stop, err := time.Parse(time.RFC3339Nano, body.Stop)
	if err != nil {
		writeTextResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if !stop.After(start) {
		msg := "invalid time range"
		writeTextResponse(w, http.StatusBadRequest, msg)
		return
	}

	digester := h.DigesterProvider(start, stop)
	digest, err := digester.Digest()
	if err != nil {
		writeTextResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer digest.Close()
	if err := h.Storage.Store(r.Context(), body.ID, digest); err != nil {
		writeTextResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	// We may want to improve this in the future to be a non-fatal error. Today if unmark fails,
	// fetching the digest will result in a perpetual "in progress" state. To mitigate this, we
	// report a failure to the caller signifying that the operation should be retried. This will
	// hopefully mitigate the amount of invalid state occurrence we may incur
	if err := h.Marker.Unmark(r.Context(), body.ID); err != nil {
		writeTextResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeTextResponse(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write([]byte(msg))
}
