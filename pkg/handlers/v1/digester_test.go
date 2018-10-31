package v1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type timeMatcher struct {
	T time.Time
}

// Matches implements matcher and matches whether or not two times are equal
// using the preferred .Equal function from the time package.
func (m *timeMatcher) Matches(x interface{}) bool {
	t, ok := x.(time.Time)
	if !ok {
		return false
	}
	return m.T.Equal(t)
}

func (m *timeMatcher) String() string {
	return "matches two time.Time instances based on the evaluation of time.Equal()"
}

func TestPostInvalidStart(t *testing.T) {
	start := ""
	stop := time.Now().Format(time.RFC3339Nano)
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start)
	q.Set("stop", stop)
	r.URL.RawQuery = q.Encode()

	h := DigesterHandler{}
	h.Post(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestPostInvalidStop(t *testing.T) {
	start := time.Now().Format(time.RFC3339Nano)
	stop := ""
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start)
	q.Set("stop", stop)
	r.URL.RawQuery = q.Encode()

	h := DigesterHandler{}
	h.Post(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestPostInvalidRange(t *testing.T) {
	start := time.Now().Format(time.RFC3339Nano)
	stop := time.Now().Add(-1 * time.Minute).Format(time.RFC3339Nano)
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start)
	q.Set("stop", stop)
	r.URL.RawQuery = q.Encode()

	h := DigesterHandler{}
	h.Post(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestPostConflictInProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	start := time.Now().Format(time.RFC3339Nano)
	stop := time.Now().Format(time.RFC3339Nano)
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start)
	q.Set("stop", stop)
	r.URL.RawQuery = q.Encode()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, &types.ErrInProgress{})

	h := DigesterHandler{
		Storage: storageMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusConflict, w.Result().StatusCode)
}

func TestPostConflictDigestCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	start := time.Now().Format(time.RFC3339Nano)
	stop := time.Now().Format(time.RFC3339Nano)
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start)
	q.Set("stop", stop)
	r.URL.RawQuery = q.Encode()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil)

	h := DigesterHandler{
		Storage: storageMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusConflict, w.Result().StatusCode)
}

func TestPostStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	start := time.Now().Format(time.RFC3339Nano)
	stop := time.Now().Format(time.RFC3339Nano)
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start)
	q.Set("stop", stop)
	r.URL.RawQuery = q.Encode()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, errors.New("oops"))

	h := DigesterHandler{
		Storage: storageMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestPostQueueError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	start := time.Now()
	stop := time.Now()
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start.Format(time.RFC3339Nano))
	q.Set("stop", stop.Format(time.RFC3339Nano))
	r.URL.RawQuery = q.Encode()

	expectedStart := &timeMatcher{start.Truncate(time.Minute)}
	expectedStop := &timeMatcher{stop.Truncate(time.Minute)}

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
	queuerMock := NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), gomock.Any(), expectedStart, expectedStop).Return(errors.New("oops"))

	h := DigesterHandler{
		Storage: storageMock,
		Queuer:  queuerMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestPostHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	start := time.Now()
	stop := time.Now()
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start.Format(time.RFC3339Nano))
	q.Set("stop", stop.Format(time.RFC3339Nano))
	r.URL.RawQuery = q.Encode()

	expectedStart := &timeMatcher{start.Truncate(time.Minute)}
	expectedStop := &timeMatcher{stop.Truncate(time.Minute)}

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
	queuerMock := NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), gomock.Any(), expectedStart, expectedStop).Return(nil)

	h := DigesterHandler{
		Storage: storageMock,
		Queuer:  queuerMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)
}
