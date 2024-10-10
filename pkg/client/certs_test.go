package client_test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/reddec/syno-cli/pkg/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListCerts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	syno := client.New(client.FromEnv(environ()))
	list, err := syno.ListCerts(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, list)
	log.Println(list)
}

func TestClient_UploadCert(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ca, err := os.Open(filepath.Join("../../test-data", "rootCA.pem"))
	require.NoError(t, err)
	defer ca.Close()

	key, err := os.Open(filepath.Join("../../test-data", "example.com-key.pem"))
	require.NoError(t, err)
	defer key.Close()

	cert, err := os.Open(filepath.Join("../../test-data", "example.com.pem"))
	require.NoError(t, err)
	defer cert.Close()

	syno := client.New(client.FromEnv(environ()))

	info, err := syno.UploadCert(ctx, client.NewCertificate{
		Name: "example.com",
		Cert: cert,
		CA:   ca,
		Key:  key,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, info.CertificateID)
}

func TestClient_DeleteCertByID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ca, err := os.Open(filepath.Join("../../test-data", "rootCA.pem"))
	require.NoError(t, err)
	defer ca.Close()

	key, err := os.Open(filepath.Join("../../test-data", "example.com-key.pem"))
	require.NoError(t, err)
	defer key.Close()

	cert, err := os.Open(filepath.Join("../../test-data", "example.com.pem"))
	require.NoError(t, err)
	defer cert.Close()

	syno := client.New(client.FromEnv(environ()))

	info, err := syno.UploadCert(ctx, client.NewCertificate{
		Name: "example.com",
		Cert: cert,
		CA:   ca,
		Key:  key,
	})
	require.NoError(t, err)

	_, err = syno.DeleteCertByID(ctx, info.CertificateID)
	require.NoError(t, err)

	// check that removed
	list, err := syno.ListCerts(ctx)
	require.NoError(t, err)
	for _, item := range list {
		assert.NotEqual(t, info.CertificateID, item.ID)
	}
}
