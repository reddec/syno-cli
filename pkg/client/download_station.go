package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// https://global.download.synology.com/download/Document/Software/DeveloperGuide/Package/DownloadStation/All/enu/Synology_Download_Station_Web_API.pdf

type DownloadTask struct {
	URL           []string  // HTTP/FTP/magnet/ED2K links or the file path starting with a shared folder.
	File          io.Reader // Optional. File (ex: torrent) uploading from client
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

// DownloadStation API
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
	if task.File != nil {
		// it must go last
		params = append(params, field{Name: "file", Value: task.File})
	}
	res, err := ds.cl.directCall(ctx, `SYNO.DownloadStation.Task`, `create`, params)
	if err != nil {
		return fmt.Errorf("call API: %w", err)
	}
	defer res.Body.Close()
	io.Copy(os.Stderr, res.Body)
	return nil
}

func setIfNotEmpty(store []field, name string, value string) []field {
	if len(value) > 0 {
		return append(store, field{Name: name, Value: value})
	}
	return store
}
