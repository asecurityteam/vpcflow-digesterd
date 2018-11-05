package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/handlers/v1"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/storage"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/stream"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
	storageBucket := mustEnv("DIGEST_STORAGE_BUCKET")
	progressBucket := mustEnv("DIGEST_PROGRESS_BUCKET")
	streamApplianceEndpoint := mustEnv("STREAM_APPLIANCE_ENDPOINT")
	streamApplianceTopic := mustEnv("STREAM_APPLIANCE_TOPIC")
	streamApplianceURL, err := url.Parse(streamApplianceEndpoint)
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
	router := chi.NewRouter()
	router.Post("/", digesterHandler.Post)
	router.Get("/", digesterHandler.Get)

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	s := &http.Server{
		Addr:    ":" + port,
		Handler: router,
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
