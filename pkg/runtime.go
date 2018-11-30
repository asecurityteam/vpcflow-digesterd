package digesterd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"bitbucket.org/atlassian/go-vpcflow"
	"bitbucket.org/atlassian/transport"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/handlers/v1"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/storage"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/stream"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/go-chi/chi"
)

// Server is an interface for starting/stopping an HTTP server
type Server interface {
	// ListenAndServe starts the HTTP server in a blocking call.
	ListenAndServe() error
	// Shutdown stops the server from accepting new connections.
	// If the given context expires before shutdown is complete then
	// the context error is returned.
	Shutdown(ctx context.Context) error
}

// Service is a container for all of the pluggable modules used by the service
type Service struct {
	// HTTPClient is the client to be used with the default Queuer module.
	// If no client is provided, a default will be used.
	HTTPClient *http.Client

	// Queuer is responsible for queuing digester jobs which will eventually be consumed
	// by the Produce handler. The built in Queuer POSTs to an HTTP endpoint.
	Queuer types.Queuer

	// Storage provides a mechanism to hook into a persistent store for the digests. The
	// built in Storage uses S3 as the persistent storage for digest blobs.
	Storage types.Storage

	// Marker is responsible for marking which digests jobs are inprogress. The built in
	// Marker uses S3 to hold this state.
	Marker types.Marker
}

func (s *Service) init() error {
	var err error
	storageClient, err := createS3Client(mustEnv("DIGEST_STORAGE_BUCKET_REGION"))
	if err != nil {
		return err
	}
	progressClient, err := createS3Client(mustEnv("DIGEST_PROGRESS_BUCKET"))
	if err != nil {
		return err
	}

	if s.ErrorCallback == nil {
		s.ErrorCallback = func(_ context.Context, _ int, _ error) {}
	}
	if s.Queuer == nil {
		streamApplianceEndpoint := mustEnv("STREAM_APPLIANCE_ENDPOINT")
		streamApplianceURL, err := url.Parse(streamApplianceEndpoint)
		if err != nil {
			return err
		}
		if s.HTTPClient == nil {
			retrier := transport.NewRetrier(
				transport.NewFixedBackoffPolicy(50*time.Millisecond),
				transport.NewLimitedRetryPolicy(3),
				transport.NewStatusCodeRetryPolicy(500, 502, 503),
			)
			base := transport.NewFactory(
				transport.OptionDefaultTransport,
				transport.OptionDisableCompression(true),
				transport.OptionTLSHandshakeTimeout(time.Second),
				transport.OptionMaxIdleConns(100),
			)
			recycler := transport.NewRecycler(
				transport.Chain{retrier}.ApplyFactory(base),
				transport.RecycleOptionTTL(10*time.Minute),
				transport.RecycleOptionTTLJitter(time.Minute),
			)
			s.HTTPClient = &http.Client{Transport: recycler}
		}
		s.Queuer = &stream.DigestQueuer{
			Client:   s.HTTPClient,
			Endpoint: streamApplianceURL,
			Topic:    mustEnv("STREAM_APPLIANCE_TOPIC"),
		}
	}
	if s.Storage == nil {
		s.Storage = &storage.InProgress{
			Bucket: mustEnv("DIGEST_PROGRESS_BUCKET"),
			Client: progressClient,
			Storage: &storage.S3{
				Bucket: mustEnv("DIGEST_STORAGE_BUCKET"),
				Client: storageClient,
			},
		}
	}
	if s.Marker == nil {
		s.Marker = &storage.ProgressMarker{
			Bucket: mustEnv("DIGEST_PROGRESS_BUCKET"),
			Client: progressClient,
		}
	}
	return nil
}

// BindRoutes binds the service handlers to the provided router
func (s *Service) BindRoutes(router chi.Router) error {
	if err := s.init(); err != nil {
		return err
	}
	vpcflowBucket := mustEnv("VPC_FLOW_LOGS_BUCKET")
	vpcflowRegion := mustEnv("VPC_FLOW_LOGS_BUCKET_REGION")
	maxBytesPrefetch := mustEnv("VPC_MAX_BYTES_PREFETCH")
	maxConcurrentPrefetch := mustEnv("VPC_MAX_CONCURRENT_PREFETCH")
	maxBytes, err := strconv.ParseInt(maxBytesPrefetch, 10, 64)
	if err != nil {
		return err
	}
	maxConcurrent, err := strconv.Atoi(maxConcurrentPrefetch)
	if err != nil {
		return err
	}
	s3Client, err := createS3Client(vpcflowRegion)
	if err != nil {
		return err
	}
	digesterHandler := &v1.DigesterHandler{
		Queuer:  s.Queuer,
		Storage: s.Storage,
		Marker:  s.Marker,
	}
	produceHandler := &v1.Produce{
		Storage:          s.Storage,
		Marker:           s.Marker,
		DigesterProvider: newDigester(vpcflowBucket, s3Client, maxBytes, maxConcurrent),
	}
	router.Post("/", digesterHandler.Post)
	router.Get("/", digesterHandler.Get)
	router.Post("/{topic}/{event}", produceHandler.ServeHTTP)
	return nil
}

// Runtime is the app configuration and execution point
type Runtime struct {
	Server      Server
	ExitSignals types.ExitSignals
}

// Run runs the application
func (r *Runtime) Run() error {
	exit := make(chan error)

	for _, c := range r.ExitSignals {
		go func(c chan error) {
			exit <- <-c
		}(c)
	}

	go func() {
		exit <- r.Server.ListenAndServe()
	}()

	err := <-exit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = r.Server.Shutdown(ctx)

	return err
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("%s is required", key))
	}
	return val
}

func createS3Client(region string) (*s3.S3, error) {
	useIAM := mustEnv("USE_IAM")
	useIAMFlag, err := strconv.ParseBool(useIAM)
	if err != nil {
		return nil, err
	}
	cfg := aws.NewConfig()
	cfg.Region = aws.String(region)
	if !useIAMFlag {
		cfg.Credentials = credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{
				Filename: os.Getenv("AWS_CREDENTIALS_FILE"),
				Profile:  os.Getenv("AWS_CREDENTIALS_PROFILE"),
			},
		})
	}
	awsSession, err := session.NewSession(cfg)
	if err != nil {
		return nil, err
	}
	return s3.New(awsSession), nil
}

func newDigester(bucket string, client s3iface.S3API, maxBytes int64, concurrency int) types.DigesterProvider {
	return func(start, stop time.Time) vpcflow.Digester {
		bucketIter := &vpcflow.BucketStateIterator{
			Bucket: bucket,
			Queue:  client,
		}
		filterIter := &vpcflow.BucketFilter{
			BucketIterator: bucketIter,
			Filter: vpcflow.LogFileTimeFilter{
				Start: start,
				End:   stop,
			},
		}
		readerIter := &vpcflow.BucketIteratorReader{
			BucketIterator: filterIter,
			FetchPolicy:    vpcflow.NewPrefetchPolicy(client, maxBytes, concurrency),
		}
		return &vpcflow.ReaderDigester{Reader: readerIter}
	}
}
