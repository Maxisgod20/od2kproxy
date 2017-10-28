package od2kproxy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"net/http/httptest"

	"encoding/json"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {

	viper.SetConfigType("json")

	var config = []byte(`{
		"http_timeout": 60,
		"username": "testuser",
		"password": "secret"
	}`)

	viper.ReadConfig(bytes.NewBuffer(config))

	code := m.Run()
	os.Exit(code)
}

func TestNewProxyClientConfig(t *testing.T) {
	assert := assert.New(t)

	client, err := NewProxyClient()
	assert.Nil(err)
	assert.Equal(time.Second*time.Duration(60), client.client.Timeout)
	assert.Equal("testuser", client.username)
	assert.Equal("secret", client.password)
}

func TestProxyClientbuildURL(t *testing.T) {
	assert := assert.New(t)

	client, err := NewProxyClient()
	assert.Nil(err)

	// no query string
	url := client.buildURL("/resources/1", "")
	assert.Equal("https://gegevensmagazijn.tweedekamer.nl/resources/1", url)

	// with query string
	url = client.buildURL("/resources/1", "a=1&b=2")
	assert.Equal("https://gegevensmagazijn.tweedekamer.nl/resources/1?a=1&b=2", url)
}

// Tests data not compressed as no encoding is set by client
func TestHandlerUncompressed(t *testing.T) {
	assert := assert.New(t)

	// mock server of external location
	body := []byte(`uncompressed`)
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(body)
		}))
	defer ts.Close()

	var configMap = map[string]interface{}{
		"http_timeout": 2,
		"username":     "testuser",
		"password":     "secret",
	}
	jsonConfig, err := json.Marshal(configMap)
	viper.ReadConfig(bytes.NewBuffer(jsonConfig))

	proxy, err := NewProxyClient()
	// mock proxy client url
	proxy.baseURL = ts.URL
	assert.Nil(err)

	// request to proxy without gzip encoding
	testURL := fmt.Sprintf("%s/foo/1", ts.URL)
	clientRequest, _ := http.NewRequest("GET", testURL, nil)
	w := httptest.NewRecorder()
	proxy.Handler(w, clientRequest)

	// compare received headers from proxy
	assert.Equal("12", w.HeaderMap.Get("Content-Length"))
	assert.Equal("text/plain; charset=utf-8", w.HeaderMap.Get("Content-Type"))

	result := string(w.Body.Bytes()[:])
	assert.Equal("uncompressed", result)
	assert.Equal(200, w.Code)
}

// Tests data compression as gzip encoding requested by client
func TestHandlerRequestGzip(t *testing.T) {
	assert := assert.New(t)

	// mock server of external location
	body := []byte(`aaaa`)
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(body)
		}))
	defer ts.Close()

	var configMap = map[string]interface{}{
		"http_timeout": 2,
		"username":     "testuser",
		"password":     "secret",
	}
	jsonConfig, err := json.Marshal(configMap)
	viper.ReadConfig(bytes.NewBuffer(jsonConfig))

	proxy, err := NewProxyClient()
	// mock proxy client url
	proxy.baseURL = ts.URL
	assert.Nil(err)

	// request to proxy without gzip encoding
	testURL := fmt.Sprintf("%s/foo/1", ts.URL)
	clientRequest, _ := http.NewRequest("GET", testURL, nil)
	clientRequest.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	proxy.Handler(w, clientRequest)

	// compare received headers from proxy
	assert.Equal("gzip", w.HeaderMap.Get("Content-Encoding"))
	assert.Equal("28", w.HeaderMap.Get("Content-Length"))
	assert.Equal("text/plain; charset=utf-8", w.HeaderMap.Get("Content-Type"))

	// gzip the mock body for comparison
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(body)
	gz.Close()
	assert.Equal(buf.Bytes(), w.Body.Bytes())
	assert.Equal(200, w.Code)
}

// Tests data not gzipped when response headers of the proxied call already contains gzip encoded
// data.
func TestHandlerCompressed(t *testing.T) {
	assert := assert.New(t)

	var configMap = map[string]interface{}{
		"http_timeout": 2,
		"username":     "testuser",
		"password":     "secret",
	}
	jsonConfig, err := json.Marshal(configMap)
	viper.ReadConfig(bytes.NewBuffer(jsonConfig))

	// mock server of external location
	body := []byte(`{"data": already gzipped}`)
	bodyLen := fmt.Sprintf("%d", len(body))
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", bodyLen)
			w.Write(body)
		}),
	)
	defer mockServer.Close()

	proxy, err := NewProxyClient()
	// mock proxy client url
	proxy.baseURL = mockServer.URL
	assert.Nil(err)

	// request to proxy without gzip encoding
	testURL := fmt.Sprintf("%s/foo/1", mockServer.URL)
	clientRequest, _ := http.NewRequest("GET", testURL, nil)
	w := httptest.NewRecorder()
	proxy.Handler(w, clientRequest)

	assert.Equal("gzip", w.HeaderMap.Get("Content-Encoding"))
	assert.Equal(bodyLen, w.HeaderMap.Get("Content-Length"))
	assert.Equal("application/json", w.HeaderMap.Get("Content-Type"))
	// compare the gzipped proxied response
	assert.Equal(body, w.Body.Bytes())
	assert.Equal(200, w.Code)
}

// Tests NewProxyClient returning error when username and password are missing.
// This test must be run last, because it overrides the viper config
func TestNewProxyClientBrokenConfig(t *testing.T) {
	assert := assert.New(t)

	// empty config
	var config = []byte(`{}`)
	viper.ReadConfig(bytes.NewBuffer(config))
	client, err := NewProxyClient()

	assert.Nil(client)
	assert.EqualError(err, "username is required")

	// http_protocol set
	viper.ReadConfig(bytes.NewBuffer([]byte(`{
		"username": "testuser"
	}`)))
	client, err = NewProxyClient()
	assert.Nil(client)
	assert.EqualError(err, "password is required")
}

func TestHandlerTimeout(t *testing.T) {
	assert := assert.New(t)

	// mock server of external location
	body := []byte(`aaaa`)
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second * time.Duration(2))
			w.Write(body)
		}))
	defer ts.Close()

	var configMap = map[string]interface{}{
		"http_timeout": 1,
		"username":     "testuser",
		"password":     "secret",
	}
	jsonConfig, err := json.Marshal(configMap)
	viper.ReadConfig(bytes.NewBuffer(jsonConfig))

	client, err := NewProxyClient()
	// mock proxy client url
	client.baseURL = ts.URL
	assert.Nil(err)

	// request to proxy without gzip encoding
	testURL := fmt.Sprintf("%s/foo/1", ts.URL)
	req, _ := http.NewRequest("GET", testURL, nil)
	w := httptest.NewRecorder()
	client.Handler(w, req)

	expectedHeaders := http.Header{}
	assert.Equal(expectedHeaders, w.HeaderMap)

	result := string(w.Body.Bytes()[:])
	assert.Equal("Proxy Timeout", result)
	assert.Equal(504, w.Code)
}
