package storage

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
)

// ProgressMarker is an implementation of Marker which allows for marking/unmarking of digests in progress
type ProgressMarker struct {
	Bucket   string
	Client   s3iface.S3API
	uploader s3manageriface.UploaderAPI
	lock     sync.Mutex
	now      func() time.Time
}

// Mark flags the digest identified by key as being "in progress"
func (m *ProgressMarker) Mark(ctx context.Context, key string) error {
	m.initUploader()
	now := m.now
	if now == nil {
		now = time.Now
	}
	_, err := m.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(m.Bucket),
		Key:    aws.String(key + inProgressSuffix),
		Body:   bytes.NewReader([]byte(now().Format(time.RFC3339Nano))),
	})
	return err
}

// Unmark flags the digest identified by key as not being "in progress"
func (m *ProgressMarker) Unmark(ctx context.Context, key string) error {
	_, err := m.Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(m.Bucket),
		Key:    aws.String(key + inProgressSuffix),
	})
	return err
}

func (m *ProgressMarker) initUploader() {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.uploader == nil {
		m.uploader = s3manager.NewUploaderWithClient(m.Client)
	}
}
