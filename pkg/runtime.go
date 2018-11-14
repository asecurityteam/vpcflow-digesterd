package digesterd

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
	S3Client s3iface.S3API
	Queuer   types.Queuer
	Storage  types.Storage
	Marker   types.Marker
}

func (s *Service) init() error {
	var err error
	if s.S3Client == nil {
		s.S3Client, err = createS3Client()
		if err != nil {
			return err
		}
	}
	if s.Queuer == nil {
		streamApplianceEndpoint := mustEnv("STREAM_APPLIANCE_ENDPOINT")
		streamApplianceURL, err := url.Parse(streamApplianceEndpoint)
		if err != nil {
			return err
		}
		s.Queuer = &stream.DigestQueuer{
			Client:   &http.Client{},
			Endpoint: streamApplianceURL,
			Topic:    mustEnv("STREAM_APPLIANCE_TOPIC"),
		}
	}
	if s.Storage == nil {
		s.Storage = &storage.InProgress{
			Bucket: mustEnv("DIGEST_PROGRESS_BUCKET"),
			Client: s.S3Client,
			Storage: &storage.S3{
				Bucket: mustEnv("DIGEST_STORAGE_BUCKET"),
				Client: s.S3Client,
			},
		}
	}
	if s.Marker == nil {
		s.Marker = &storage.ProgressMarker{
			Bucket: mustEnv("DIGEST_PROGRESS_BUCKET"),
			Client: s.S3Client,
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
	digesterHandler := &v1.DigesterHandler{
		Queuer:  s.Queuer,
		Storage: s.Storage,
		Marker:  s.Marker,
	}
	produceHandler := &v1.Produce{
		Storage:          s.Storage,
		Marker:           s.Marker,
		DigesterProvider: newDigester(vpcflowBucket, s.S3Client, maxBytes, maxConcurrent),
	}
	router.Post("/", digesterHandler.Post)
	router.Get("/", digesterHandler.Get)
	router.Post("/{topic}/{event}", produceHandler.ServeHTTP)
	return nil
}

// Runtime is the app configuration and execution point
type Runtime struct {
	Server Server
}

// Run runs the application
func (r *Runtime) Run() error {
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	go func() {
		if err := r.Server.ListenAndServe(); err != nil {
			log.Fatal(err.Error())
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = r.Server.Shutdown(ctx)

	return nil
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("%s is required", key))
	}
	return val
}

func createS3Client() (*s3.S3, error) {
	region := mustEnv("REGION")
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
