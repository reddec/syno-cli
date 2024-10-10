package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	EnvURL  = "SYNOLOGY_URL"
	EnvUser = "SYNOLOGY_USER"
	EnvPass = "SYNOLOGY_PASSWORD" //nolint:gosec
)

const DefaultTimeout = 30 * time.Second

var ErrBadStatus = errors.New("bad response status")

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPClientFunc func(req *http.Request) (*http.Response, error)

func (hf HTTPClientFunc) Do(req *http.Request) (*http.Response, error) {
	return hf(req)
}

type Config struct {
	Client   HTTPClient // HTTP client to perform requests, default is new HTTP client. Client MUST support cookies. Keep it nil for most cases is a good idea.
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
		jar, err := cookiejar.New(nil)
		if err != nil {
			panic(err) // CAN NOT happen at go 1.23
		}
		cfg.Client = &http.Client{
			Jar:     jar,
			Timeout: DefaultTimeout,
		}
	}
	if cfg.URL == "" {
		cfg.URL = "http://localhost:5000"
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
	authorized  atomic.Bool
	authLock    sync.Mutex
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
	if cl.authorized.Load() {
		return nil
	}

	cl.authLock.Lock()
	defer cl.authLock.Unlock()
	if cl.authorized.Load() {
		return nil
	}

	res, err := cl.directCall(ctx, "SYNO.API.Auth", "login", []field{
		{Name: "enable_syno_token", Value: "no"},
		{Name: "account", Value: cl.user},
		{Name: "passwd", Value: cl.password},
		{Name: "format", Value: "cookie"},
	})
	if err != nil {
		return fmt.Errorf("invoke api: %w", err)
	}

	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	cl.authorized.Store(true)
	return nil
}

// DownloadStation API
func (cl *Client) DownloadStation() *DownloadStation {
	return &DownloadStation{cl: cl}
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

// deprecated, use directCall instead
func (cl *Client) doPost(ctx context.Context, path string, queryParams map[string]interface{}, params map[string]interface{}, out interface{}) error {
	var contentType string
	var content io.ReadCloser
	if needStreaming(params) {
		contentType, content = streamData(mapToFields(params))
	} else {
		contentType, content = plainData(mapToFields(params))
	}
	defer content.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cl.baseURL+path+"?"+joinParams(mapToFields(queryParams)), content)
	if err != nil {
		return fmt.Errorf("prepare request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

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

func (cl *Client) directCall(ctx context.Context, apiName string, method string, params []field) (*http.Response, error) {
	info, err := cl.APIVersion(ctx, apiName)
	if err != nil {
		return nil, fmt.Errorf("get API %s version: %w", apiName, err)
	}

	params = append([]field{
		{Name: "api", Value: apiName},
		{Name: "version", Value: info.MaxVersion},
		{Name: "method", Value: method},
	}, params...)
	requestURL := cl.baseURL + "/webapi/" + info.Path + "/" + apiName

	var contentType string
	var content io.ReadCloser
	if needStreamingIter(params) {
		contentType, content = streamData(params)
	} else {
		contentType, content = plainData(params)
	}
	defer content.Close()

	slog.Debug("API request prepared", "url", requestURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, content)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	res, err := cl.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call API: %w", err)
	}
	defer res.Body.Close()

	//nolint:mnd
	if res.StatusCode/100 != 2 {
		_ = res.Body.Close()
		return nil, fmt.Errorf("status %d: %w", res.StatusCode, ErrBadStatus)
	}

	// try to parse body as API response
	var buffer bytes.Buffer
	if err := asAPIError(io.TeeReader(res.Body, &buffer)); err != nil {
		_ = res.Body.Close()
		return nil, fmt.Errorf("application API error: %w", err)
	}
	res.Body = &readCloser{
		Reader: io.MultiReader(&buffer, res.Body),
		Closer: res.Body,
	}

	return res, nil
}

func asAPIError(data io.Reader) error {
	var rawResponse apiResponse

	err := json.NewDecoder(data).Decode(&rawResponse)
	if err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !rawResponse.Success {
		return rawResponse.Error
	}
	return nil
}

func plainData(params []field) (string, io.ReadCloser) {
	return "application/x-www-form-urlencoded", io.NopCloser(strings.NewReader(joinParams(params)))
}

func streamData(fields []field) (string, io.ReadCloser) {
	reader, writer := io.Pipe()
	mp := multipart.NewWriter(writer)

	go func() {
		err := streamMultipart(fields, mp)
		if err == nil {
			err = mp.Close()
		}
		_ = writer.CloseWithError(err)
	}()
	return mp.FormDataContentType(), reader
}

func joinParams(params []field) string {
	var buffer bytes.Buffer
	for _, field := range params {
		if buffer.Len() > 0 {
			buffer.WriteRune('&')
		}
		buffer.WriteString(url.QueryEscape(field.Name))
		buffer.WriteRune('=')
		if v, ok := field.Value.([]byte); ok {
			buffer.WriteString(url.QueryEscape(string(v)))
		} else {
			buffer.WriteString(url.QueryEscape(fmt.Sprint(field.Value)))
		}
	}
	return buffer.String()
}

type field struct {
	Name  string
	Value interface{} // Reader, fileAttachment, []byte, string, else (Sprint'able)
}

func streamMultipart(fields []field, w *multipart.Writer) error {
	for _, field := range fields {
		var dest io.Writer
		var source io.Reader
		switch v := field.Value.(type) {
		case io.Reader:
			out, err := w.CreateFormField(field.Name)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", field.Name, err)
			}
			dest = out
			source = v
		case fileAttachment:
			out, err := w.CreateFormFile(field.Name, v.FileName)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", field.Name, err)
			}
			dest = out
			source = v.Reader
		case []byte:
			out, err := w.CreateFormField(field.Name)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", field.Name, err)
			}
			dest = out
			source = bytes.NewReader(v)
		case string:
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", `form-data; name=`+strconv.Quote(field.Name))

			out, err := w.CreatePart(h)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", field.Name, err)
			}
			dest = out
			source = strings.NewReader(v)
		default:
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", `form-data; name=`+strconv.Quote(field.Name))
			out, err := w.CreatePart(h)
			if err != nil {
				return fmt.Errorf("create part for %s: %w", field.Name, err)
			}
			dest = out
			source = strings.NewReader(fmt.Sprint(v))
		}
		if _, err := io.Copy(dest, source); err != nil {
			return fmt.Errorf("copy content for part %s: %w", field.Name, err)
		}
	}
	return nil
}

// deprecated
func needStreaming(params map[string]interface{}) bool {
	for _, v := range params {
		switch v.(type) {
		case io.Reader, *fileAttachment, fileAttachment:
			return true
		}
	}
	return false
}

func needStreamingIter(params []field) bool {
	for _, f := range params {
		switch f.Value.(type) {
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
	return "API error code: " + strconv.FormatInt(e.Code, 10)
}

type readCloser struct {
	io.Reader
	io.Closer
}

// deprecated, used for compatibility only
func mapToFields(fields map[string]interface{}) []field {
	l := make([]field, 0, len(fields))
	for k, v := range fields {
		l = append(l, field{
			Name:  k,
			Value: v,
		})
	}
	return l
}
