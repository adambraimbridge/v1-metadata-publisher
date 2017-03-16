package metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"fmt"
	"os"

	"io/ioutil"

	"github.com/stretchr/testify/assert"
)

const UUID = "0cd42702-f789-11e6-9516-2d969e0d3b65"

var UUIDResponse = []byte("{ \"uuid\" : \"0cd42702-f789-11e6-9516-2d969e0d3b65\", \"identifiers\" : [{ \"authority\" : \"http://api.ft.com/system/FTCOM-METHODE\" }] } \n")

type MockMetadataReadService struct {
	mockReadByUUID func(content Content) ([]byte, error)
}

func (mr *MockMetadataReadService) ReadByUUID(content Content) ([]byte, error) {
	return mr.mockReadByUUID(content)
}

type MockContentService struct {
	mockSaveContent func(file string) (*os.File, int, error)
}

func (cs *MockContentService) SaveContent(file string) (*os.File, int, error) {
	return cs.SaveContent(file)

}

func TestPublishMetadataForUUIDSuccesfully(t *testing.T) {
	ps := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "binding-service", r.Header.Get("X-Origin-System-Id"), "Invalid X-Origin-System-Id header value")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Invalid Content-Type header value")
		w.Header().Add("X-Request-Id", "tid_testtid")
	}))
	defer ps.Close()

	mps := V1MetadataPublishService{
		publishing: &Cluster{
			address:  ps.URL + "/__cms-metadata-notifier/notify",
			username: "foo",
			password: "bar",
		},
	}

	metadata, err := getMetadata()
	assert.NoError(t, err, "Failed to read metadata")
	err = mps.publishMetadataForUUID("0cd42702-f789-11e6-9516-2d969e0d3b65", metadata)
	assert.NoError(t, err, "Failed to publish metadata")
}

func TestPublishMetadataForUUIDUnsuccesfully(t *testing.T) {
	ps := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ps.Close()

	mps := V1MetadataPublishService{
		publishing: &Cluster{
			address:  ps.URL + "/__cms-metadata-notifier/notify",
			username: "foo",
			password: "bar",
		},
	}

	metadata, err := getMetadata()
	assert.NoError(t, err, "Failed to read metadata")
	err = mps.publishMetadataForUUID("0cd42702-f789-11e6-9516-2d969e0d3b65", metadata)
	assert.Error(t, err, "Expected metadata publish will return an error")
}

func TestSendMetadataJobSuccessfully(t *testing.T) {
	ps := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "binding-service", r.Header.Get("X-Origin-System-Id"), "Invalid X-Origin-System-Id header value")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Invalid Content-Type header value")
		w.Header().Add("X-Request-Id", "tid_testtid")
	}))
	defer ps.Close()

	mps := V1MetadataPublishService{
		publishing: &Cluster{
			address:  ps.URL + "/__cms-metadata-notifier/notify",
			username: "foo",
			password: "bar",
		},
		mr: &MockMetadataReadService{
			mockReadByUUID: func(content Content) ([]byte, error) {
				return getMetadata()
			},
		},
	}

	errorsCh := make(chan error)
	doneCh := make(chan bool)
	contents := []Content{
		{
			UUID:        "0cd42702-f789-11e6-9516-2d969e0d3b65",
			Identifiers: []Identifier{{Authority: "http://api.ft.com/system/FTCOM-METHODE"}},
		},
	}

	go mps.sendMetadataJob(contents, errorsCh, doneCh)
	<-doneCh
	for e := range errorsCh {
		assert.NoError(t, e, "Error while publishing metadata")
	}
}

func TestSendMetadataJobNotAvailableMetadataService(t *testing.T) {
	ps := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "binding-service", r.Header.Get("X-Origin-System-Id"), "Invalid X-Origin-System-Id header value")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Invalid Content-Type header value")
		w.Header().Add("X-Request-Id", "tid_testtid")
	}))
	defer ps.Close()

	mps := V1MetadataPublishService{
		publishing: &Cluster{
			address:  ps.URL + "/__cms-metadata-notifier/notify",
			username: "foo",
			password: "bar",
		},
		mr: &MockMetadataReadService{
			mockReadByUUID: func(content Content) ([]byte, error) {
				return nil, fmt.Errorf("Cannot get metadata")
			},
		},
	}

	errorsCh := make(chan error)
	doneCh := make(chan bool)
	contents := []Content{
		{
			UUID:        "0cd42702-f789-11e6-9516-2d969e0d3b65",
			Identifiers: []Identifier{{Authority: "http://api.ft.com/system/FTCOM-METHODE"}},
		},
	}

	go mps.sendMetadataJob(contents, errorsCh, doneCh)
	<-doneCh
	for e := range errorsCh {
		assert.Error(t, e, "Error while publishing metadata")
	}
}

func TestSendMetadataJobPublishingClusterNotAvailable(t *testing.T) {
	ps := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ps.Close()

	mps := V1MetadataPublishService{
		publishing: &Cluster{
			address:  ps.URL + "/__cms-metadata-notifier/notify",
			username: "foo",
			password: "bar",
		},
		mr: &MockMetadataReadService{
			mockReadByUUID: func(content Content) ([]byte, error) {
				return nil, fmt.Errorf("Cannot get metadata")
			},
		},
	}

	errorsCh := make(chan error)
	doneCh := make(chan bool)
	contents := []Content{
		{
			UUID:        "0cd42702-f789-11e6-9516-2d969e0d3b65",
			Identifiers: []Identifier{{Authority: "http://api.ft.com/system/FTCOM-METHODE"}},
		},
	}

	go mps.sendMetadataJob(contents, errorsCh, doneCh)
	<-doneCh
	for e := range errorsCh {
		assert.Error(t, e, "Error while publishing metadata")
	}
}

func TestPublishSuccesfull(t *testing.T) {
	ps := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "binding-service", r.Header.Get("X-Origin-System-Id"), "Invalid X-Origin-System-Id header value")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Invalid Content-Type header value")
		w.Header().Add("X-Request-Id", "tid_testtid")
	}))
	defer ps.Close()

	mps := V1MetadataPublishService{
		cs: &MockContentService{
			mockSaveContent: func(file string) (*os.File, int, error) {
				err := ioutil.WriteFile(file, UUIDResponse, 065)
				assert.NoError(t, err, "")
				f, err := os.Open(file)
				return f, 2, err
			},
		},
		publishing: &Cluster{
			address:  ps.URL + "/__cms-metadata-notifier/notify",
			username: "foo",
			password: "bar",
		},
		mr: &MockMetadataReadService{
			mockReadByUUID: func(content Content) ([]byte, error) {
				return getMetadata()
			},
		},
		batchSize: 10,
		source:    "METHODE",
	}

	err := mps.Publish()
	assert.NoError(t, err, "Error while trying to publish metadata")
}