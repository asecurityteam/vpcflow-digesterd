package storage

import (
	"context"
	"io"

	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

const keySuffix = ".log.gz"

// S3Storage implements the Storage interface and uses S3 as the backing store for digests
type S3Storage struct {
	Bucket string
	Client s3iface.S3API
}

// Get returns the digest for the given key.
func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key + keySuffix),
	}
	res, err := s.Client.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, parseNotFound(err, key)
	}
	return res.Body, nil
}

// Exists returns true if the digest exists, but does not download the digest body.
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key + keySuffix),
	}
	_, err := s.Client.HeadObjectWithContext(ctx, input)
	if err == nil {
		return true, nil
	}
	if _, ok := parseNotFound(err, key).(types.ErrNotFound); ok {
		return false, nil
	}
	return false, err
}

// Store stores the digest. It is the caller's responsibility to call Close on the Reader when done.
func (s *S3Storage) Store(ctx context.Context, key string, data io.ReadCloser) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key + keySuffix),
		Body:   aws.ReadSeekCloser(data),
	}
	_, err := s.Client.PutObjectWithContext(ctx, input)
	return err
}

// If a key is not found, transform to our NotFound error, otherwise return original error
func parseNotFound(err error, key string) error {
	if aErr, ok := err.(awserr.Error); ok && aErr.Code() == s3.ErrCodeNoSuchKey {
		return types.ErrNotFound{ID: key}
	}
	return err
}
