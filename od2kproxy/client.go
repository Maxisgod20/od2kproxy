package od2kproxy

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type ProxyClient struct {
	client   *http.Client
	username string
	password string
	baseURL  string
}

// NewProxyClient creates a new instance of ProxyClient
func NewProxyClient() (*ProxyClient, error) {
	timeout := viper.GetInt("http_timeout")
	timeoutHTTP := time.Second * time.Duration(timeout)
	if timeout == 0 {
		timeoutHTTP = time.Second * time.Duration(30)
	}

	client := &http.Client{
		Timeout: timeoutHTTP,
	}

	username := viper.GetString("username")
	if username == "" {
		return nil, errors.New("username is required")
	}

	password := viper.GetString("password")
	if password == "" {
		return nil, errors.New("password is required")
	}

	baseURL := "https://gegevensmagazijn.tweedekamer.nl"

	return &ProxyClient{
		client:   client,
		username: username,
		password: password,
		baseURL:  baseURL,
	}, nil
}

// buildUrl creates the url used to request a resource at the original service
func (p ProxyClient) buildURL(path string, query string) string {
	if query == "" {
		return fmt.Sprintf("%s%s", p.baseURL, path)
	}
	return fmt.Sprintf("%s%s?%s", p.baseURL, path, query)
}

func (p ProxyClient) ErrorResponse(status string, code int, body io.ReadCloser, req *http.Request) *http.Response {
	return &http.Response{
		Status:        status,
		StatusCode:    code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          body,
		ContentLength: 0,
		Request:       req,
	}
}

// DoRequest does a http request
func (p ProxyClient) DoRequest(uri *url.URL) *http.Response {
	url := p.buildURL(uri.Path, uri.RawQuery)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		errorBody := ioutil.NopCloser(bytes.NewBufferString(err.Error()))
		return p.ErrorResponse("500 Invalid Proxy Request", 504, errorBody, req)
	}

	// Set headers
	req.SetBasicAuth(p.username, p.password)
	req.Header.Set("Accept-Encoding", "gzip")

	// Do request
	resp, err := p.client.Do(req)
	if err != nil {
		if err.(net.Error).Timeout() {
			errorBody := ioutil.NopCloser(bytes.NewBufferString("Proxy Timeout"))
			return p.ErrorResponse("504 Proxy Timeout", 504, errorBody, req)
		} else {
			errorBody := ioutil.NopCloser(bytes.NewBufferString(err.Error()))
			return p.ErrorResponse("500 Internal Proxy Error", 500, errorBody, req)
		}
	}

	log.Printf("[%d] %s", resp.StatusCode, url)
	return resp
}

// Handler acts as the proxy, doing the request and handling gzip encoding
func (p ProxyClient) Handler(w http.ResponseWriter, r *http.Request) {
	resp := p.DoRequest(r.URL)
	if resp.StatusCode > 500 {
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	} else {
		defer resp.Body.Close()

		// Forward all headers from proxied response and check for gzipped content
		gzipped := false
		for key := range resp.Header {
			value := resp.Header.Get(key)
			w.Header().Set(key, value)

			if value == "gzip" {
				gzipped = true
			}
		}

		// forward already gzipped content by proxied server
		if gzipped {
			io.Copy(w, resp.Body)
		} else {
			encoding := r.Header.Get("Accept-Encoding")
			// gzip compress the response body if client accepts gzip
			if strings.Contains(encoding, "gzip") {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				io.Copy(gz, resp.Body)
				gz.Close()

				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

				w.WriteHeader(resp.StatusCode)
				w.Write(buf.Bytes())
			} else {
				// return body normally
				io.Copy(w, resp.Body)
			}
		}
	}
}
