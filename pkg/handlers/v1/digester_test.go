package v1

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/atlassian/httplog"
	"bitbucket.org/atlassian/logevent"
	"bitbucket.org/atlassian/vpcflow-digesterd/mocks"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func logProvider(_ context.Context) logevent.Logger {
	return logevent.New(logevent.Config{Output: ioutil.Discard})
}

func logEventProvider(_ context.Context) httplog.Event {
	return httplog.Event{}
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

	h := DigesterHandler{
		LogProvider:      logProvider,
		LogEventProvider: logEventProvider,
	}
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

	h := DigesterHandler{
		LogProvider:      logProvider,
		LogEventProvider: logEventProvider,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestPostInvalidRange(t *testing.T) {
	start := time.Now().Format(time.RFC3339Nano)
	stop := time.Now().Add(-1 * time.Second).Format(time.RFC3339Nano)
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	q := r.URL.Query()
	q.Set("start", start)
	q.Set("stop", stop)
	r.URL.RawQuery = q.Encode()

	h := DigesterHandler{
		LogProvider:      logProvider,
		LogEventProvider: logEventProvider,
	}
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

	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().Get(gomock.Any()).Return(nil, &types.ErrInProgress{})

	h := DigesterHandler{
		LogProvider:      logProvider,
		LogEventProvider: logEventProvider,
		Storage:          storageMock,
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

	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().Get(gomock.Any()).Return([]byte("digest"), nil)

	h := DigesterHandler{
		LogProvider:      logProvider,
		LogEventProvider: logEventProvider,
		Storage:          storageMock,
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

	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().Get(gomock.Any()).Return(nil, errors.New("oops"))

	h := DigesterHandler{
		LogProvider:      logProvider,
		LogEventProvider: logEventProvider,
		Storage:          storageMock,
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

	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().Get(gomock.Any()).Return([]byte{}, nil)
	queuerMock := mocks.NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), start.Truncate(time.Second), stop.Truncate(time.Second)).Return(errors.New("oops"))

	h := DigesterHandler{
		LogProvider:      logProvider,
		LogEventProvider: logEventProvider,
		Storage:          storageMock,
		Queuer:           queuerMock,
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

	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().Get(gomock.Any()).Return([]byte{}, nil)
	queuerMock := mocks.NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), start.Truncate(time.Second), stop.Truncate(time.Second)).Return(nil)

	h := DigesterHandler{
		LogProvider:      logProvider,
		LogEventProvider: logEventProvider,
		Storage:          storageMock,
		Queuer:           queuerMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)
}
