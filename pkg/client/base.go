package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	EnvURL  = "SYNOLOGY_URL"
	EnvUser = "SYNOLOGY_USER"
	EnvPass = "SYNOLOGY_PASSWORD" //nolint:gosec
)

var ErrBadStatus = errors.New("bad response status")

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPClientFunc func(req *http.Request) (*http.Response, error)

func (hf HTTPClientFunc) Do(req *http.Request) (*http.Response, error) {
	return hf(req)
}

type Config struct {
	Client   HTTPClient // HTTP client to perform requests, default is http.DefaultClient
	User     string     // User name
	Password string     // User password
	URL      string     // Synology url, default is http://localhost:5000
}

// Default client based on env variables.
func Default() *Client {
	return New(FromEnv(nil))
}

// FromEnv creates config based on standard environment variables. If envFunc not defined,
// os.Getenv will be used.
func FromEnv(envFunc func(string) string) Config {
	if envFunc == nil {
		envFunc = os.Getenv
	}

	return Config{
		User:     envFunc(EnvUser),
		Password: envFunc(EnvPass),
		URL:      envFunc(EnvURL),
	}
}

// New instance of Synology API client.
func New(cfg Config) *Client {
	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}
	if cfg.URL == "" {
		cfg.URL = "http://localhost:500"
	} else {
		cfg.URL = strings.TrimRight(cfg.URL, "/")
	}

	return &Client{
		client:   cfg.Client,
		user:     cfg.User,
		password: cfg.Password,
		baseURL:  cfg.URL,
	}
}

type Client struct {
	client      HTTPClient
	user        string
	password    string
	baseURL     string
	authLock    sync.Mutex
	token       string
	sid         string
	versionLock sync.Mutex
	versions    map[string]API
}

// WithClient returns copy of Synology client with custom HTTP client.
func (cl *Client) WithClient(client HTTPClient) *Client {
	cl.versionLock.Lock()
	defer cl.versionLock.Unlock()
	cl.authLock.Lock()
	defer cl.authLock.Unlock()

	return &Client{
		client:   client,
		user:     cl.user,
		password: cl.password,
		baseURL:  cl.baseURL,
		token:    cl.token,
		sid:      cl.sid,
		versions: cl.versions,
	}
}

// APIVersion returns max version for specific API. It queries Synology for all APIs and caches result.
func (cl *Client) APIVersion(ctx context.Context, apiName string) (API, error) {
	if m := cl.versions; m != nil {
		return m[apiName], nil
	}
	cl.versionLock.Lock()
	defer cl.versionLock.Unlock()
	if m := cl.versions; m != nil {
		return m[apiName], nil
	}

	err := cl.doPost(ctx, "/webapi/query.cgi", nil, map[string]interface{}{
		"method":  "query",
		"api":     "SYNO.API.Info",
		"version": 1,
	}, &cl.versions)
	if err != nil {
		return API{}, fmt.Errorf("invoke api: %w", err)
	}

	return cl.versions[apiName], nil
}

// Login to Synology and get token. Token will be cached. If token already obtained, API call will not be executed.
func (cl *Client) Login(ctx context.Context) error {
	if cl.token != "" {
		return nil
	}

	cl.authLock.Lock()
	defer cl.authLock.Unlock()
	if cl.token != "" {
		return nil
	}

	var response struct {
		Sid   string `json:"sid"`
		Token string `json:"synotoken"`
	}
	err := cl.callAPI(ctx, "SYNO.API.Auth", "login", map[string]interface{}{
		"enable_syno_token": "yes",
		"account":           cl.user,
		"passwd":            cl.password,
	}, &response)
	if err != nil {
		return fmt.Errorf("invoke api: %w", err)
	}
	cl.token = response.Token
	cl.sid = response.Sid
	return nil
}

func (cl *Client) callAPI(ctx context.Context, apiName, method string, params map[string]interface{}, out interface{}) error {
	info, err := cl.APIVersion(ctx, apiName)
	if err != nil {
		return fmt.Errorf("get API %s version: %w", apiName, err)
	}

	var queryParams = map[string]interface{}{
		"method":  method,
		"api":     apiName,
		"version": info.MaxVersion,
		"_sid":    cl.sid,
	}

	// if it's not upload, we can merge transport params into payload
	if !needStreaming(params) {
		if params == nil {
			params = queryParams
		} else {
			for k, v := range queryParams {
				params[k] = v
			}
			queryParams = nil
		}
	}

	return cl.doPost(ctx, "/webapi/"+info.Path, queryParams, params, out)
}

func (cl *Client) doPost(ctx context.Context, path string, queryParams map[string]interface{}, params map[string]interface{}, out interface{}) error {
	var contentType string
	var content io.ReadCloser
	if needStreaming(params) {
		contentType, content = streamData(params)
	} else {
		contentType, content = plainData(params)
	}
	defer content.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cl.baseURL+path+"?"+joinParams(queryParams), content)
	if err != nil {
		return fmt.Errorf("prepare request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Syno-Token", cl.token)

	res, err := cl.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %w", res.StatusCode, ErrBadStatus)
	}

	var rawResponse apiResponse

	err = json.NewDecoder(res.Body).Decode(&rawResponse)
	if err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if err := rawResponse.Error; err != nil {
		return fmt.Errorf("response: %w", err)
	}
	err = json.Unmarshal(rawResponse.RawData, out)
	if err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	return nil
}

func plainData(params map[string]interface{}) (string, io.ReadCloser) {
	return "application/x-www-form-urlencoded", io.NopCloser(strings.NewReader(joinParams(params)))
}

func streamData(params map[string]interface{}) (string, io.ReadCloser) {
	reader, writer := io.Pipe()
	mp := multipart.NewWriter(writer)

	go func() {
		err := streamMultipart(params, mp)
		if err == nil {
			err = mp.Close()
		}
		_ = writer.CloseWithError(err)
	}()

	return "multipart/form-data; boundary=" + mp.Boundary(), reader
}

func joinParams(params map[string]interface{}) string {
	var buffer bytes.Buffer
	for key, value := range params {
		if buffer.Len() > 0 {
			buffer.WriteRune('&')
		}
		buffer.WriteString(url.QueryEscape(key))
		buffer.WriteRune('=')
		buffer.WriteString(url.QueryEscape(fmt.Sprint(value)))
	}
	return buffer.String()
}

func streamMultipart(params map[string]interface{}, w *multipart.Writer) error {
	for key, value := range params {
		var dest io.Writer
		var source io.Reader
		switch v := value.(type) {
		case io.Reader:
			out, err := w.CreateFormField(key)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", key, err)
			}
			dest = out
			source = v
		case fileAttachment:
			out, err := w.CreateFormFile(key, v.FileName)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", key, err)
			}
			dest = out
			source = v.Reader
		case []byte:
			out, err := w.CreateFormField(key)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", key, err)
			}
			dest = out
			source = bytes.NewReader(v)
		case string:
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", `form-data; name=`+strconv.Quote(key))
			h.Set("Content-Type", "text/plain")

			out, err := w.CreatePart(h)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", key, err)
			}
			dest = out
			source = strings.NewReader(v)
		default:
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", `form-data; name=`+strconv.Quote(key))
			out, err := w.CreatePart(h)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", key, err)
			}
			dest = out
			source = strings.NewReader(fmt.Sprint(v))
		}
		if _, err := io.Copy(dest, source); err != nil {
			return fmt.Errorf("copy content for part %s: %w", key, err)
		}
	}
	return nil
}

func needStreaming(params map[string]interface{}) bool {
	for _, v := range params {
		switch v.(type) {
		case io.Reader, *fileAttachment, fileAttachment:
			return true
		}
	}
	return false
}

type fileAttachment struct {
	FileName string
	Reader   io.Reader
}

type apiResponse struct {
	Success bool            `json:"success"`
	Error   *RemoteError    `json:"error,omitempty"`
	RawData json.RawMessage `json:"data"`
}

type API struct {
	MaxVersion int64  `json:"maxVersion"`
	Path       string `json:"path"`
}

type RemoteError struct {
	Code int64 `json:"code"`
}

func (e *RemoteError) Error() string {
	return "API error code: " + strconv.FormatInt(e.Code, 10) //nolint:gomnd
}
