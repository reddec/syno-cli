package commands

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
	"github.com/reddec/syno-cli/pkg/client"
)

//nolint:staticcheck
type CertsAuto struct {
	SynoClient  `group:"Synology Client" namespace:"synology" env-namespace:"SYNOLOGY"`
	CacheDir    string        `short:"c" long:"cache-dir" env:"CACHE_DIR" description:"Cache location for accounts information" default:".cache"`
	RenewBefore time.Duration `short:"r" long:"renew-before" env:"RENEW_BEFORE" description:"Renew certificate time reserve" default:"720h"`
	Email       string        `short:"e" long:"email" env:"EMAIL" description:"Email for contact"`
	Provider    string        `short:"p" long:"provider" env:"PROVIDER" description:"DNS challenge provider" required:"true" `
	DNS         []string      `short:"D" long:"dns" env:"DNS" env-delim:","  description:"Custom resolvers" default:"8.8.8.8"`
	Timeout     time.Duration `short:"t" long:"timeout" env:"TIMEOUT" description:"DNS challenge timeout" default:"1m"`
	Domains     []string      `short:"d" long:"domains" env:"DOMAINS" env-delim:","  description:"Domains names to issue" required:"true"`
}

func (lc *CertsAuto) Execute([]string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	account, err := lc.getOrCreateAccount()
	if err != nil {
		return err
	}

	log.Println("setting challenger for", lc.Provider)
	if err := lc.setupChallenge(account); err != nil {
		return err
	}

	log.Println("start initial setup")

	for {
		log.Println("issuing or renewing certificates if needed")
		if list, err := lc.issueOrRenewCerts(ctx, account); err != nil {
			log.Println("failed issue certs:", err)
		} else if err := lc.pushToSynology(ctx, list); err != nil {
			log.Println("failed push to Synology:", err)
		}
		log.Println("done, next check after 1 hour")
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Hour):
		}
	}
}

func (lc *CertsAuto) issueOrRenewCerts(ctx context.Context, lgc *lego.Client) ([]*certificate.Resource, error) {
	var certs []*certificate.Resource
	for _, domain := range lc.Domains {
		cert, err := loadCert(lc.CacheDir, domain)
		if errors.Is(err, os.ErrNotExist) {
			log.Println("issuing new certificate for domain", domain)
			// issue
			cert, err = lc.issueCert(domain, lgc)
			if err != nil {
				return nil, fmt.Errorf("issue new certificate for %s: %w", domain, err)
			}
		} else if err != nil {
			// something happen during load
			return certs, err
		} else if crt, err := parseCert(cert); err != nil {
			// parsing failed
			return certs, err
		} else if time.Now().After(crt.NotAfter) {
			log.Println("issuing new certificate for domain", domain, "because the old one expired")
			// expired
			// issue
			cert, err = lc.issueCert(domain, lgc)
			if err != nil {
				return nil, fmt.Errorf("issue new certificate for %s: %w", domain, err)
			}
		} else if time.Until(crt.NotAfter) <= lc.RenewBefore {
			// renew if soon expired
			cert, err = lc.renewCert(cert, lgc)
			if err != nil {
				return nil, fmt.Errorf("issue new certificate for %s: %w", domain, err)
			}
		}
		certs = append(certs, cert)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	return certs, nil
}

func (lc *CertsAuto) pushToSynology(ctx context.Context, certs []*certificate.Resource) error {
	syno := lc.SynoClient.Client()

	knownCerts, err := syno.ListCerts(ctx)
	if err != nil {
		return err
	}

	var index = make(map[string]client.Certificate)
	for _, c := range knownCerts {
		index[c.Description] = c
	}

	for _, res := range certs {
		log.Println("pushing certs of", res.Domain, "to Synology")
		status, err := syno.UploadCert(ctx, client.NewCertificate{
			Name: res.Domain,
			Cert: bytes.NewReader(res.Certificate),
			CA:   bytes.NewReader(res.IssuerCertificate),
			Key:  bytes.NewReader(res.PrivateKey),
		})
		if err != nil {
			return fmt.Errorf("push to synology for domain %s: %w", res.Domain, err)
		}
		log.Println("certificate ID:", status.CertificateID, "server restarted:", status.ServerRestarted)
	}

	return nil
}

func (lc *CertsAuto) issueCert(domain string, lgc *lego.Client) (*certificate.Resource, error) {
	request, err := lgc.Certificate.Obtain(certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("create certificate request for: %w", err)
	}
	return request, saveCert(lc.CacheDir, request)
}

func (lc *CertsAuto) renewCert(res *certificate.Resource, lgc *lego.Client) (*certificate.Resource, error) {
	ng, err := lgc.Certificate.Renew(*res, true, false, "")
	if err != nil {
		return nil, err
	}
	return ng, saveCert(lc.CacheDir, ng)
}

func (lc *CertsAuto) setupChallenge(lgc *lego.Client) error {
	provider, err := dns.NewDNSChallengeProviderByName(lc.Provider)
	if err != nil {
		return err
	}

	var opts = []dns01.ChallengeOption{
		dns01.AddDNSTimeout(lc.Timeout),
	}
	if len(lc.DNS) > 0 {
		opts = append(opts, dns01.AddRecursiveNameservers(lc.DNS))
	}
	return lgc.Challenge.SetDNS01Provider(provider, opts...)
}

func (lc *CertsAuto) getOrCreateAccount() (*lego.Client, error) {
	accountFile := filepath.Join(lc.CacheDir, lc.Email+".json")
	account, err := loadAccount(accountFile)
	if err == nil {
		log.Println("using saved account")
		return lego.NewClient(lego.NewConfig(account))
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	log.Println("generating new account")

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	user := &legoAccount{
		Email: lc.Email,
		Key:   privateKey,
	}

	config := lego.NewConfig(user)

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}

	user.Registration = reg

	return client, user.Save(accountFile)
}

type legoAccount struct {
	Email        string
	Registration *registration.Resource
	Key          *ecdsa.PrivateKey `json:"-"`
}

func (la *legoAccount) GetEmail() string {
	return la.Email
}

func (la *legoAccount) GetRegistration() *registration.Resource {
	return la.Registration
}

func (la *legoAccount) GetPrivateKey() crypto.PrivateKey {
	return la.Key
}

func (la *legoAccount) MarshalJSON() ([]byte, error) {
	data, err := x509.MarshalECPrivateKey(la.Key)
	if err != nil {
		return nil, err
	}

	return json.Marshal(serializedLegoAccount{
		Email:        la.Email,
		Registration: la.Registration,
		RawKey:       data,
	})
}

func (la *legoAccount) UnmarshalJSON(bytes []byte) error {
	var acc serializedLegoAccount
	err := json.Unmarshal(bytes, &acc)
	if err != nil {
		return err
	}

	key, err := x509.ParseECPrivateKey(acc.RawKey)
	if err != nil {
		return err
	}
	la.Email = acc.Email
	la.Key = key
	la.Registration = acc.Registration
	return nil
}

func (la *legoAccount) Save(file string) error {
	return atomicJSON(file, la)
}

func loadAccount(file string) (*legoAccount, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var acc legoAccount

	return &acc, json.NewDecoder(f).Decode(&acc)
}

type serializedLegoAccount struct {
	Email        string
	Registration *registration.Resource
	RawKey       []byte
}

func saveCert(dir string, resource *certificate.Resource) error {
	return atomicJSON(filepath.Join(dir, resource.Domain+".json"), serializedCertificate{
		Domain:            resource.Domain,
		CertURL:           resource.CertURL,
		CertStableURL:     resource.CertStableURL,
		PrivateKey:        resource.PrivateKey,
		Certificate:       resource.Certificate,
		IssuerCertificate: resource.IssuerCertificate,
		CSR:               resource.CSR,
	})
}

func loadCert(dir, domain string) (*certificate.Resource, error) {
	f, err := os.Open(filepath.Join(dir, domain+".json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var resource serializedCertificate
	err = json.NewDecoder(f).Decode(&resource)
	if err != nil {
		return nil, err
	}

	return &certificate.Resource{
		Domain:            resource.Domain,
		CertURL:           resource.CertURL,
		CertStableURL:     resource.CertStableURL,
		PrivateKey:        resource.PrivateKey,
		Certificate:       resource.Certificate,
		IssuerCertificate: resource.IssuerCertificate,
		CSR:               resource.CSR,
	}, nil
}

type serializedCertificate struct {
	Domain            string `json:"domain"`
	CertURL           string `json:"certUrl"`
	CertStableURL     string `json:"certStableUrl"`
	PrivateKey        []byte `json:"privateKey"`
	Certificate       []byte `json:"certificate"`
	IssuerCertificate []byte `json:"issuer_certificate"`
	CSR               []byte `json:"csr"`
}

func atomicJSON(file string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}
	tempFile := file + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer os.Remove(tempFile)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(data)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return os.Rename(tempFile, file)
}

func parseCert(cert *certificate.Resource) (*x509.Certificate, error) {
	info, _ := pem.Decode(cert.Certificate)
	return x509.ParseCertificate(info.Bytes)
}
