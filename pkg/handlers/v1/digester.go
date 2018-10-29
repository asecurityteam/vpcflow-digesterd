package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	gosrvhttp "bitbucket.org/atlassian/gosrv/pkg/http"
	"bitbucket.org/atlassian/httplog"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/satori/go.uuid"
)

const (
	unknownObject = "unknown"
	actionCreate  = "create"
)

// DigesterHandler handles incoming HTTP requests for starting and retrieving new digests
type DigesterHandler struct {
	LogProvider      types.LoggerProvider
	LogEventProvider types.LogEventProvider
	Storage          types.Storage
	Queuer           types.Queuer
}

// Post creates a new digest
func (h *DigesterHandler) Post(w http.ResponseWriter, r *http.Request) {
	logger := h.LogProvider(r.Context())
	start, stop, err := extractInput(r)
	if err != nil {
		logger.Info(h.newDigestEvent(r.Context(), http.StatusBadRequest, unknownObject, err.Error(), actionCreate))
		gosrvhttp.JSONErrors(w, http.StatusBadRequest, err)
		return
	}
	id := computeID(start, stop)
	data, err := h.Storage.Get(id)
	switch err.(type) {
	case nil:
	case *types.ErrInProgress:
		logger.Info(h.newDigestEvent(r.Context(), http.StatusConflict, id, err.Error(), actionCreate))
		gosrvhttp.JSONErrors(w, http.StatusConflict, err)
		return
	default:
		logger.Error(h.newDigestEvent(r.Context(), http.StatusInternalServerError, id, err.Error(), actionCreate))
		gosrvhttp.JSONMessages(w, http.StatusInternalServerError, "internal")
		return
	}
	// if data is returned, a digest already exists. return 409 and exit
	if len(data) > 0 {
		msg := fmt.Sprintf("digest %s already exists", id)
		logger.Info(h.newDigestEvent(r.Context(), http.StatusConflict, id, msg, actionCreate))
		gosrvhttp.JSONMessages(w, http.StatusConflict, msg)
		return
	}

	if err := h.Queuer.Queue(id, start, stop); err != nil {
		logger.Error(h.newDigestEvent(r.Context(), http.StatusInternalServerError, id, err.Error(), actionCreate))
		gosrvhttp.JSONMessages(w, http.StatusInternalServerError, "internal")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// extractInput attempts to extract the start/stop query parameters required by GET and POST.
// If either value is not a valid RFC3339Nano, an error is returned. Otherwise, start and stop
// times are returned in the respective order. Additionally, it truncates the time values to the
// nearest second since anything with more precision doesn't really fit the digest filter use case
func extractInput(r *http.Request) (time.Time, time.Time, error) {
	startString := r.URL.Query().Get("start")
	stopString := r.URL.Query().Get("stop")
	start, err := time.Parse(time.RFC3339Nano, startString)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	stop, err := time.Parse(time.RFC3339Nano, stopString)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if start.After(stop) {
		return time.Time{}, time.Time{}, errors.New("start should be before stop")
	}
	return start.Truncate(time.Second), stop.Truncate(time.Second), nil
}

// computeID generates a UUID v5 from a name composed by appending start and stop time strings
// in that order
func computeID(start, stop time.Time) string {
	name := start.String() + stop.String()
	u := uuid.NewV5(uuid.NamespaceURL, name)
	return u.String()
}

// return a new Event prepopulated with data from the context and the given fields
func (h *DigesterHandler) newDigestEvent(ctx context.Context, status int, objectID, result, action string) httplog.Event {
	e := h.LogEventProvider(ctx)
	e.Status = status
	e.ObjectID = objectID
	e.ObjectType = "digest"
	e.Result = result
	e.Action = action
	return e
}
