package digesterd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterSlice(t *testing.T) {
	tc := []struct {
		Name   string
		Input  []string
		Output []string
	}{
		{
			Name:   "empty_slice",
			Input:  []string{},
			Output: []string{},
		},
		{
			Name:   "filtered_slice",
			Input:  []string{"", "a", "", "b", "", "c", ""},
			Output: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Output, filterSlice(tt.Input))
		})
	}
}

func TestMakePrefix(t *testing.T) {
	tc := []struct {
		Name     string
		Regions  []string
		Accounts []string
		Date     time.Time
		Prefix   string
	}{
		{
			Name:     "no_region",
			Regions:  []string{},
			Accounts: []string{"a"},
			Date:     time.Now(),
			Prefix:   "",
		},
		{
			Name:     "no_account",
			Regions:  []string{"r"},
			Accounts: []string{},
			Date:     time.Now(),
			Prefix:   "",
		},
		{
			Name:     "single_digits",
			Regions:  []string{"r"},
			Accounts: []string{"a"},
			Date:     time.Date(2019, time.January, 1, 0, 0, 0, 0, time.Local),
			Prefix:   "AWSLogs/a/vpcflowlogs/r/2019/01/01",
		},
		{
			Name:     "double_digits",
			Regions:  []string{"r"},
			Accounts: []string{"a"},
			Date:     time.Date(2019, time.October, 12, 0, 0, 0, 0, time.Local),
			Prefix:   "AWSLogs/a/vpcflowlogs/r/2019/10/12",
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Prefix, makePrefix(tt.Regions, tt.Accounts, tt.Date))
		})
	}
}

func TestNewDigesterSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Client := NewMockS3API(ctrl)
	digesterFunc := newDigester("bucket", mockS3Client, 64, 1, []string{"region"}, []string{"accounts"})
	digester := digesterFunc(time.Time{}, time.Time{})
	require.NotNil(t, digester)
}

func TestMustEnv(t *testing.T) {
	tc := []struct {
		Name  string
		Key   string
		Value string
		Set   bool
	}{
		{
			Name:  "var unset",
			Key:   "key",
			Value: "",
			Set:   false,
		},
		{
			Name:  "var set",
			Key:   "key",
			Value: "value",
			Set:   true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			originalValue, existing := os.LookupEnv(tt.Key)
			if existing {
				defer os.Setenv(tt.Key, originalValue)
			}
			os.Unsetenv(tt.Key)
			if tt.Set {
				os.Setenv(tt.Key, tt.Value)
				require.Equal(t, tt.Value, mustEnv(tt.Key))
				return
			}
			require.Panics(t, func() {
				mustEnv(tt.Key)
			})

		})
	}
}

func TestServiceInitSuccess(t *testing.T) {
	// save current environment variables, and restore them
	// after the test ends
	environ := os.Environ()
	os.Clearenv()
	defer func() {
		for _, e := range environ {
			envPair := strings.Split(e, "=")
			os.Setenv(envPair[0], envPair[1])
		}
	}()

	// set required test environment variables
	os.Setenv("USE_IAM", "true")
	os.Setenv("DIGEST_STORAGE_BUCKET_REGION", "n/a")
	os.Setenv("DIGEST_PROGRESS_BUCKET_REGION", "n/a")
	os.Setenv("STREAM_APPLIANCE_ENDPOINT", "n/a")
	os.Setenv("DIGEST_PROGRESS_TIMEOUT", "1")
	os.Setenv("DIGEST_PROGRESS_BUCKET", "n/a")
	os.Setenv("DIGEST_STORAGE_BUCKET", "n/a")
	s := &Service{}
	require.Nil(t, s.init())
}

func TestServiceBindRoutesSuccess(t *testing.T) {
	environ := os.Environ()
	os.Clearenv()
	defer func() {
		for _, e := range environ {
			envPair := strings.Split(e, "=")
			os.Setenv(envPair[0], envPair[1])
		}
	}()

	// set required test environment variables
	os.Setenv("USE_IAM", "true")
	os.Setenv("DIGEST_STORAGE_BUCKET_REGION", "n/a")
	os.Setenv("DIGEST_PROGRESS_BUCKET_REGION", "n/a")
	os.Setenv("STREAM_APPLIANCE_ENDPOINT", "n/a")
	os.Setenv("DIGEST_PROGRESS_TIMEOUT", "1")
	os.Setenv("DIGEST_PROGRESS_BUCKET", "n/a")
	os.Setenv("DIGEST_STORAGE_BUCKET", "n/a")
	os.Setenv("VPC_FLOW_LOGS_BUCKET", "n/a")
	os.Setenv("VPC_FLOW_LOGS_BUCKET_REGION", "n/a")
	os.Setenv("VPC_MAX_BYTES_PREFETCH", "1")
	os.Setenv("VPC_MAX_CONCURRENT_PREFETCH", "1")
	os.Setenv("VPC_FLOW_LOGS_SCAN_REGIONS", "n/a")
	os.Setenv("VPC_FLOW_LOGS_SCAN_REGIONS", "n/a")

	router := chi.NewMux()
	s := &Service{}
	require.Nil(t, s.BindRoutes(router))
}
