package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type FileType string

// Supported file types.
// Extracted from 'allowfilters' in JS (download.js)
const (
	FileTypeUnknown FileType = ""
	FileTypeTorrent FileType = "torrent"
	FileTypeNZB     FileType = "nzb"
	FileTypeTxt     FileType = "txt"
)

var ErrUnknownFileType = errors.New("unknown file type")

const peekSize = 512 // should be enough for XML header, unless intentionally obfuscated

type DownloadTask struct {
	URL           []string  // HTTP/FTP/magnet/ED2K links or the file path starting with a shared folder.
	File          io.Reader // Optional. File (ex: torrent) uploading from client
	FileType      FileType  // Optional. Type of File. If not set, it will try automatically detect (which may fail, so better set it).
	Username      string    // Optional. Login username for remote resource (not NAS!)
	Password      string    // Optional. Login password for remote resource (not NAS!)
	UnzipPassword string    // Optional. Password for unzipping download tasks
	Destination   string    // Optional. Download destination path starting with a shared folder
}

type DownloadTasks struct {
	Total  int64 `json:"total"`
	Offset int64 `json:"offset"`
	Tasks  []struct {
		Id         string `json:"id"`
		Type       string `json:"type"`
		Username   string `json:"username"`
		Title      string `json:"title"`
		Size       int64  `json:"size"`
		Status     string `json:"status"`
		Additional struct {
			Detail struct {
				CreateTime  int64  `json:"create_time"`
				Destination string `json:"destination"`
				Priority    string `json:"priority"`
				Uri         string `json:"uri"`
			} `json:"detail"`
		} `json:"additional"`
	} `json:"tasks"`
}

// DownloadStation API. Enhanced by some undocumented API from JS.
//
// See https://global.download.synology.com/download/Document/Software/DeveloperGuide/Package/DownloadStation/All/enu/Synology_Download_Station_Web_API.pdf
type DownloadStation struct {
	cl *Client
}

// List all download tasks in NAS. Implies 'detail' feature. Limit -1 means all.
func (ds *DownloadStation) List(ctx context.Context, offset, limit int) (*DownloadTasks, error) {
	if err := ds.cl.Login(ctx); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	res, err := ds.cl.directCall(ctx, `SYNO.DownloadStation.Task`, `list`, []field{
		{Name: "offset", Value: offset},
		{Name: "limit", Value: limit},
		{Name: "additional", Value: "detail"},
	})
	if err != nil {
		return nil, fmt.Errorf("call: %w", err)
	}
	defer res.Body.Close()
	var response struct {
		Data DownloadTasks `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &response.Data, nil
}

// Download remote data from HTTP/FTP/magnet/ED2K links or the file path starting with a shared folder.
// This is simplified version of Create.
func (ds *DownloadStation) Download(ctx context.Context, destination string, urls ...string) error {
	return ds.Create(ctx, DownloadTask{
		URL:         urls,
		Destination: destination,
	})
}

// Create download task in DownloadStation based on configuration.
func (ds *DownloadStation) Create(ctx context.Context, task DownloadTask) error {
	if task.File != nil {
		return ds.createV2(ctx, task)
	}
	return ds.createV1(ctx, task)
}

// download task in DownloadStation based on configuration.  Uses Synology legacy API
func (ds *DownloadStation) createV1(ctx context.Context, task DownloadTask) error {
	if err := ds.cl.Login(ctx); err != nil {
		return fmt.Errorf("login: %w", err)
	}
	var params []field
	if len(task.URL) > 0 {
		params = append(params, field{Name: "uri", Value: strings.Join(task.URL, ",")})
	}
	params = setIfNotEmpty(params, "destination", task.Destination)
	params = setIfNotEmpty(params, "username", task.Username)
	params = setIfNotEmpty(params, "password", task.Password)
	params = setIfNotEmpty(params, "unzip_password", task.UnzipPassword)
	res, err := ds.cl.directCall(ctx, `SYNO.DownloadStation.Task`, `create`, params)
	if err != nil {
		return fmt.Errorf("call API: %w", err)
	}
	defer res.Body.Close()
	return nil
}

// download task in DownloadStation based on configuration. Uses Synology2 API.
// Applies only for file uploads.
func (ds *DownloadStation) createV2(ctx context.Context, task DownloadTask) error {
	if err := ds.cl.Login(ctx); err != nil {
		return fmt.Errorf("login: %w", err)
	}
	var params []field
	params = setIfNotEmpty(params, "username", task.Username)
	params = setIfNotEmpty(params, "password", task.Password)
	if task.Destination != "" {
		value, err := json.Marshal(task.Destination)
		if err != nil {
			return fmt.Errorf("marshal destination: %w", err)
		}
		params = setIfNotEmpty(params, "destination", string(value))
	}
	params = setIfNotEmpty(params, "unzip_password", task.UnzipPassword)
	params = append(params, field{Name: "type", Value: `"file"`})

	if task.FileType == FileTypeUnknown {
		// detect by sniffing the payload (as best as we can; we can just a few)
		peek := make([]byte, peekSize)
		n, err := io.ReadFull(task.File, peek)
		if errors.Is(err, io.ErrUnexpectedEOF) {
			err = nil // nps
		}
		if err != nil {
			return fmt.Errorf("peek file: %w", err)
		}
		peek = peek[:n]

		ft, err := detectType(peek)
		if err != nil {
			return fmt.Errorf("detect file type: %w", err)
		}
		slog.Debug("filetype automatically detected", "filetype", ft)
		task.FileType = ft
		task.File = io.MultiReader(bytes.NewReader(peek), task.File)
	}
	params = append(params, field{Name: "create_list", Value: `false`}) // required
	params = append(params, field{Name: "file", Value: `["` + task.FileType + `"]`})
	params = append(params, field{Name: string(task.FileType), Value: fileAttachment{
		FileName: generateFileName() + "." + string(task.FileType),
		Reader:   task.File,
	}})
	res, err := ds.cl.directCall(ctx, `SYNO.DownloadStation2.Task`, `create`, params)
	if err != nil {
		return fmt.Errorf("call API: %w", err)
	}
	defer res.Body.Close()
	return nil
}

func setIfNotEmpty(store []field, name string, value string) []field {
	if len(value) > 0 {
		return append(store, field{Name: name, Value: value})
	}
	return store
}

func detectType(peek []byte) (FileType, error) {
	// it does it's best to detect  but no guarantees.

	switch {
	case bytes.HasPrefix(peek, []byte("d8:announce")):
		return FileTypeTorrent, nil
	case bytes.Contains(peek, []byte("<nzb")) || bytes.Contains(peek, []byte(":nzb")): // targeting xml tag nzb with xmlns or with ns
		return FileTypeNZB, nil
	case hasAnyPrefix(peek, "http://", "https://", "ftp://", "thunder://", "flashget://", "qqdl://", "magnet:"):
		// assuming this is file with list of urls
		return FileTypeTxt, nil
	}
	return FileTypeUnknown, ErrUnknownFileType
}

func hasAnyPrefix(chunk []byte, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if bytes.HasPrefix(chunk, []byte(prefix)) {
			return true
		}
	}
	return false
}

func generateFileName() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
