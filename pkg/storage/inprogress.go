package storage

import (
	"context"
	"io"
	"sync"

	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
)

const inProgressSuffix = "_in_progress"

// InProgress is an implementation of Storage which is intended to decorate the S3 implementation.
//
// The decorator will check if a digest is in progress, and if so, will return types.ErrInProgress.
// On a successful Store operation, the decorator will remove the digest's "in progress" status.
type InProgress struct {
	Bucket   string
	Client   s3iface.S3API
	uploader s3manageriface.UploaderAPI
	lock     sync.Mutex
	types.Storage
}

// Get returns the digest for the given key.
//
// If the digest is in the process of being created, an error will be returned of type types.ErrInProgress
func (s *InProgress) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	err := s.checkInProgress(ctx, key)
	// digest is in progress
	if err == nil {
		return nil, types.ErrInProgress{Key: key}
	}
	// unknown error
	if _, ok := parseNotFound(err, key).(types.ErrNotFound); !ok {
		return nil, err
	}
	return s.Storage.Get(ctx, key)
}

// Exists returns true if the digest exists, but does not download the digest body.
//
// If the digest is in the process of being created, an error will be returned of type types.ErrInProgress
func (s *InProgress) Exists(ctx context.Context, key string) (bool, error) {
	err := s.checkInProgress(ctx, key)
	// digest is in progress
	if err == nil {
		return false, types.ErrInProgress{Key: key}
	}
	// unknown error
	if _, ok := parseNotFound(err, key).(types.ErrNotFound); !ok {
		return false, err
	}
	return s.Storage.Exists(ctx, key)
}

// Store stores the digest
//
// If the operation is successful, it will remove the in progress status of the digest
func (s *InProgress) Store(ctx context.Context, key string, data io.ReadCloser) error {
	if err := s.Storage.Store(ctx, key, data); err != nil {
		return err
	}
	_, err := s.Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key + inProgressSuffix),
	})
	return err
}

// Mark flags the digest identified by key as being "in progress"
func (s *InProgress) Mark(ctx context.Context, key string) error {
	if s.uploader == nil {
		s.initUploader()
	}
	_, err := s.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key + inProgressSuffix),
	})
	return err
}

func (s *InProgress) initUploader() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.uploader == nil {
		s.uploader = s3manager.NewUploaderWithClient(s.Client)
	}
}

func (s *InProgress) checkInProgress(ctx context.Context, key string) error {
	_, err := s.Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key + inProgressSuffix),
	})
	return err
}
