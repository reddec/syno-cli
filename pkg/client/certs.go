package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Issuer struct {
	CommonName   string `json:"common_name"`
	Country      string `json:"country"`
	Organization string `json:"organization"`
}

type Service struct {
	DisplayName     string `json:"display_name"`
	DisplayNameI18N string `json:"display_name_i18n,omitempty"`
	IsPkg           bool   `json:"isPkg"`
	Owner           string `json:"owner"`
	Service         string `json:"service"`
	Subscriber      string `json:"subscriber"`
	MultipleCert    bool   `json:"multiple_cert,omitempty"`
	UserSetable     bool   `json:"user_setable,omitempty"`
}

type Subject struct {
	CommonName string   `json:"common_name"`
	SubAltName []string `json:"sub_alt_name"`
}

type Certificate struct {
	ID                 string    `json:"id"`
	Description        string    `json:"desc"`
	IsBroken           bool      `json:"is_broken"`
	IsDefault          bool      `json:"is_default"`
	Issuer             Issuer    `json:"issuer"`
	KeyTypes           string    `json:"key_types"`
	Renewable          bool      `json:"renewable"`
	Services           []Service `json:"services"`
	SignatureAlgorithm string    `json:"signature_algorithm"`
	Subject            Subject   `json:"subject"`
	UserDeletable      bool      `json:"user_deletable"`
	ValidFrom          CTime     `json:"valid_from"`
	ValidTill          CTime     `json:"valid_till"`
}

func (ct *Certificate) Expired() bool {
	return time.Now().After(ct.ValidTill.Time())
}

func (cl *Client) ListCerts(ctx context.Context) ([]Certificate, error) {
	if err := cl.Login(ctx); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	var response struct {
		Certificates []Certificate `json:"certificates"`
	}

	if err := cl.callAPI(ctx, "SYNO.Core.Certificate.CRT", "list", nil, &response); err != nil {
		return nil, fmt.Errorf("call api: %w", err)
	}

	return response.Certificates, nil
}

type NewCertificate struct {
	Name      string    // unique logical name for certificate
	AsDefault bool      // use certificate as default
	Cert      io.Reader // PEM certificate
	CA        io.Reader // optional
	Key       io.Reader // PEM private key
}

// UploadCert uploads certificate to Synology. Replaces if name (used field description) already exists.
func (cl *Client) UploadCert(ctx context.Context, draft NewCertificate) (*CertUploadResult, error) {
	var info CertUploadResult
	list, err := cl.ListCerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}
	// find if already exists
	var id string
	for _, crt := range list {
		if crt.Description == draft.Name {
			id = crt.ID
			break
		}
	}
	params := map[string]interface{}{
		"key": fileAttachment{
			FileName: "server.key",
			Reader:   draft.Key,
		},
		"cert": fileAttachment{
			FileName: "server.crt",
			Reader:   draft.Cert,
		},
		"desc": draft.Name,
	}
	if draft.CA != nil {
		params["inter_cert"] = fileAttachment{
			FileName: "ca.crt",
			Reader:   draft.CA,
		}
	}
	if draft.AsDefault {
		params["as_default"] = "true"
	}
	if id != "" {
		params["id"] = id
	}
	return &info, cl.callAPI(ctx, "SYNO.Core.Certificate", "import", params, &info)
}

// DeleteCertByID deletes certificate by known ID (not name).
func (cl *Client) DeleteCertByID(ctx context.Context, id string) (*ServerStatus, error) {
	var info ServerStatus
	if err := cl.Login(ctx); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	ids, err := json.Marshal([]string{id})
	if err != nil {
		return nil, fmt.Errorf("marshal ids: %w", err)
	}
	return &info, cl.callAPI(ctx, "SYNO.Core.Certificate.CRT", "delete", map[string]interface{}{
		"ids": string(ids),
	}, &info)
}

type CertUploadResult struct {
	CertificateID string `json:"id"`
	ServerStatus
}

type ServerStatus struct {
	ServerRestarted bool `json:"restart_httpd"`
}
