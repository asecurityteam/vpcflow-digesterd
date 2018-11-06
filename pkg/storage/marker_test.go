package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMarkInProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedInput := &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key + "_in_progress"),
	}

	mockUploader := NewMockUploaderAPI(ctrl)
	mockUploader.EXPECT().UploadWithContext(gomock.Any(), expectedInput).Return(nil, nil)

	m := &ProgressMarker{
		Bucket:   bucket,
		uploader: mockUploader,
	}

	err := m.Mark(context.Background(), key)
	assert.Nil(t, err)
}

func TestMarkInProgressError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedInput := &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key + "_in_progress"),
	}

	mockUploader := NewMockUploaderAPI(ctrl)
	mockUploader.EXPECT().UploadWithContext(gomock.Any(), expectedInput).Return(nil, errors.New("oops"))

	m := &ProgressMarker{
		Bucket:   bucket,
		uploader: mockUploader,
	}

	err := m.Mark(context.Background(), key)
	assert.NotNil(t, err)
}

func TestUnmarkInProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedInput := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key + "_in_progress"),
	}

	mockClient := NewMockS3API(ctrl)
	mockClient.EXPECT().DeleteObjectWithContext(gomock.Any(), expectedInput).Return(nil, nil)

	m := &ProgressMarker{
		Bucket: bucket,
		Client: mockClient,
	}

	err := m.Unmark(context.Background(), key)
	assert.Nil(t, err)
}

func TestUnmarkInProgressError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedInput := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key + "_in_progress"),
	}

	mockClient := NewMockS3API(ctrl)
	mockClient.EXPECT().DeleteObjectWithContext(gomock.Any(), expectedInput).Return(nil, errors.New("oops"))

	m := &ProgressMarker{
		Bucket: bucket,
		Client: mockClient,
	}

	err := m.Unmark(context.Background(), key)
	assert.NotNil(t, err)
}
