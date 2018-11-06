package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"bitbucket.org/atlassian/go-vpcflow"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/handlers/v1"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/storage"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/stream"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/go-chi/chi"
)

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("%s is required", key))
	}
	return val
}

func main() {
	port := mustEnv("PORT")
	region := mustEnv("REGION")
	vpcflowBucket := mustEnv("VPC_FLOW_LOGS_BUCKET")
	maxBytesPrefetch := mustEnv("VPC_MAX_BYTES_PREFETCH")
	maxConcurrentPrefetch := mustEnv("VPC_MAX_CONCURRENT_PREFETCH")
	storageBucket := mustEnv("DIGEST_STORAGE_BUCKET")
	progressBucket := mustEnv("DIGEST_PROGRESS_BUCKET")
	streamApplianceEndpoint := mustEnv("STREAM_APPLIANCE_ENDPOINT")
	streamApplianceTopic := mustEnv("STREAM_APPLIANCE_TOPIC")
	streamApplianceURL, err := url.Parse(streamApplianceEndpoint)
	if err != nil {
		panic(err.Error())
	}
	maxBytes, err := strconv.ParseInt(maxBytesPrefetch, 10, 64)
	if err != nil {
		panic(err.Error())
	}
	maxConcurrent, err := strconv.Atoi(maxConcurrentPrefetch)
	if err != nil {
		panic(err.Error())
	}

	cfg := aws.NewConfig() // TODO: set credential provider
	cfg.Region = &region
	s3Client := s3.New(session.New(cfg))
	store := &storage.InProgress{
		Bucket: progressBucket,
		Client: s3Client,
		Storage: &storage.S3{
			Bucket: storageBucket,
			Client: s3Client,
		},
	}
	marker := &storage.ProgressMarker{
		Bucket: progressBucket,
		Client: s3Client,
	}
	digestQueuer := &stream.DigestQueuer{
		Client:   &http.Client{},
		Endpoint: streamApplianceURL,
		Topic:    streamApplianceTopic,
	}
	digesterHandler := &v1.DigesterHandler{
		Queuer:  digestQueuer,
		Storage: store,
		Marker:  marker,
	}
	produceHandler := &v1.Produce{
		Storage:          store,
		Marker:           marker,
		DigesterProvider: newDigester(vpcflowBucket, s3Client, maxBytes, maxConcurrent),
	}
	digestRouter := chi.NewRouter()
	digestRouter.Post("/", digesterHandler.Post)
	digestRouter.Get("/", digesterHandler.Get)

	streamConsumer := chi.NewRouter()
	streamConsumer.Post("/", produceHandler.ServeHTTP)

	app := chi.NewRouter()
	app.Mount("/digest", digestRouter)
	app.Mount("/{topic}/{event}", streamConsumer)

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	s := &http.Server{
		Addr:    ":" + port,
		Handler: app,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err.Error())
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Shutdown(ctx)
}

func newDigester(bucket string, client s3iface.S3API, maxBytes int64, concurrency int) types.DigesterProvider {
	return func(start, stop time.Time) vpcflow.Digester {
		bucketIt := &vpcflow.BucketStateIterator{
			Bucket: bucket,
			Queue:  client,
		}
		filterIt := &vpcflow.BucketFilter{
			BucketIterator: bucketIt,
			Filter: vpcflow.LogFileTimeFilter{
				Start: start,
				End:   stop,
			},
		}
		readerIt := &vpcflow.BucketIteratorReader{
			BucketIterator: filterIt,
			FetchPolicy:    vpcflow.NewPrefetchPolicy(client, maxBytes, concurrency),
		}
		return &vpcflow.ReaderDigester{Reader: readerIt}
	}
}
