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

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
	queuerMock := NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), gomock.Any(), start.Truncate(time.Minute), stop.Truncate(time.Minute)).Return(errors.New("oops"))

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

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
	queuerMock := NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), gomock.Any(), start.Truncate(time.Minute), stop.Truncate(time.Minute)).Return(nil)

	h := DigesterHandler{
		Storage: storageMock,
		Queuer:  queuerMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)
}
