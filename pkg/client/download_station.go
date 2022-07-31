package client

import (
	"context"
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

// Download remote data from HTTP/FTP/magnet/ED2K links or the file path starting with a shared folder.
// This is simplified version of CreateDownloadTask.
func (cl *Client) Download(ctx context.Context, destination string, urls ...string) error {
	return cl.CreateDownloadTask(ctx, DownloadTask{
		URL:         urls,
		Destination: destination,
	})
}

func (cl *Client) CreateDownloadTask(ctx context.Context, task DownloadTask) error {
	if err := cl.Login(ctx); err != nil {
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
	res, err := cl.directCall(ctx, `SYNO.DownloadStation.Task`, `create`, params)
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
