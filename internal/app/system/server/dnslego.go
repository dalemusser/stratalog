// internal/app/system/server/dnslego.go
package server

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"time"

	"github.com/dalemusser/stratalog/internal/app/system/config"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	lego "github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/go-acme/lego/v4/registration"
	"go.uber.org/zap"
)

// How long before expiration we should renew.
// You can tighten/loosen this if desired.
const renewBefore = 30 * 24 * time.Hour

// leUser implements lego's User interface.
type leUser struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration,omitempty"`
	KeyPEM       []byte                 `json:"key_pem"` // serialized private key

	key crypto.PrivateKey // cached after decode
}

func (u *leUser) GetEmail() string {
	return u.Email
}

func (u *leUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *leUser) GetPrivateKey() crypto.PrivateKey {
	if u.key != nil {
		return u.key
	}
	key, err := certcrypto.ParsePEMPrivateKey(u.KeyPEM)
	if err != nil {
		zap.L().Fatal("failed to parse ACME private key", zap.Error(err))
	}
	u.key = key
	return u.key
}

// obtainOrLoadDNSCert either loads an existing cert/key from cache (if valid
// and not close to expiry), or uses lego + Route53 DNS-01 to obtain a new
// certificate for cfg.Domain. Renewals overwrite the existing cert/key.
func obtainOrLoadDNSCert(cfg *config.Config) (tls.Certificate, error) {
	cacheDir := cfg.LetsEncryptCacheDir
	if cacheDir == "" {
		cacheDir = "letsencrypt-cache"
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return tls.Certificate{}, err
	}

	certFile := filepath.Join(cacheDir, "cert.pem")
	keyFile := filepath.Join(cacheDir, "privkey.pem")
	userFile := filepath.Join(cacheDir, "user.json")

	// Fast path: try to reuse existing cert if it's not close to expiry.
	if cert, ok := tryUseExistingCert(certFile, keyFile); ok {
		return cert, nil
	}

	zap.L().Info("no valid cached cert found; requesting new DNS-01 cert via Route53",
		zap.String("domain", cfg.Domain),
		zap.String("cache_dir", cacheDir))

	// Build or load ACME user.
	user := loadOrCreateACMEUser(userFile, cfg.LetsEncryptEmail)

	// Lego config
	legoCfg := lego.NewConfig(&user)
	legoCfg.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(legoCfg)
	if err != nil {
		return tls.Certificate{}, err
	}

	// DNS provider: Route53 (uses AWS creds from env/instance role).
	dnsProv, err := route53.NewDNSProvider()
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := client.Challenge.SetDNS01Provider(dnsProv); err != nil {
		return tls.Certificate{}, err
	}

	// Registration: either reuse or register new.
	if user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{
			TermsOfServiceAgreed: true,
		})
		if err != nil {
			return tls.Certificate{}, err
		}
		user.Registration = reg
		// Persist user to disk
		if b, err := json.MarshalIndent(user, "", "  "); err == nil {
			_ = os.WriteFile(userFile, b, 0o600)
		} else {
			zap.L().Warn("failed to persist ACME user", zap.Error(err))
		}
	}

	// Request certificate for the primary domain only. SANs can be added later.
	req := certificate.ObtainRequest{
		Domains: []string{cfg.Domain},
		Bundle:  true,
	}
	certRes, err := client.Certificate.Obtain(req)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Write cert & key to disk
	if err := os.WriteFile(certFile, certRes.Certificate, 0o600); err != nil {
		return tls.Certificate{}, err
	}
	if err := os.WriteFile(keyFile, certRes.PrivateKey, 0o600); err != nil {
		return tls.Certificate{}, err
	}

	zap.L().Info("DNS-01 certificate obtained and cached",
		zap.String("cert_file", certFile),
		zap.String("key_file", keyFile),
		zap.Time("obtained_at", time.Now().UTC()))

	return tls.LoadX509KeyPair(certFile, keyFile)
}

// tryUseExistingCert attempts to load and reuse an existing cached cert.
// It returns (cert, true) if the cert exists, parses, and is not close to expiry.
// If the cert is missing, invalid, or near expiry, it logs and returns (tls.Certificate{}, false)
// so that the caller can obtain a new one.
func tryUseExistingCert(certFile, keyFile string) (tls.Certificate, bool) {
	if _, err := os.Stat(certFile); err != nil {
		// No cert cached yet.
		return tls.Certificate{}, false
	}

	data, err := os.ReadFile(certFile)
	if err != nil {
		zap.L().Warn("failed to read cached cert; will re-obtain", zap.Error(err))
		return tls.Certificate{}, false
	}

	// Parse PEM blocks and pick the first CERTIFICATE block.
	var leaf *x509.Certificate
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			zap.L().Warn("failed to parse certificate in cached cert; will re-obtain", zap.Error(err))
			return tls.Certificate{}, false
		}
		leaf = c
		break
	}

	if leaf == nil {
		zap.L().Warn("no certificate found in cached cert; will re-obtain")
		return tls.Certificate{}, false
	}

	untilExpiry := time.Until(leaf.NotAfter)
	if untilExpiry <= 0 {
		zap.L().Info("cached cert has expired; will obtain a new one",
			zap.Time("not_after", leaf.NotAfter))
		return tls.Certificate{}, false
	}
	if untilExpiry < renewBefore {
		zap.L().Info("cached cert is close to expiry; will obtain a new one",
			zap.Time("not_after", leaf.NotAfter),
			zap.Duration("time_remaining", untilExpiry),
			zap.Duration("renew_before", renewBefore))
		return tls.Certificate{}, false
	}

	// Cert is still valid and not close to expiry; reuse it.
	zap.L().Info("using existing DNS-01 certificate from cache",
		zap.String("cert_file", certFile),
		zap.Time("not_after", leaf.NotAfter),
		zap.Duration("time_remaining", untilExpiry))

	loaded, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		zap.L().Warn("failed to load cached cert/key pair; will re-obtain", zap.Error(err))
		return tls.Certificate{}, false
	}
	return loaded, true
}

// loadOrCreateACMEUser loads an existing ACME user from disk (if present),
// or creates a new one with a freshly generated private key.
func loadOrCreateACMEUser(userFile, email string) leUser {
	var user leUser
	if data, err := os.ReadFile(userFile); err == nil {
		if err := json.Unmarshal(data, &user); err != nil {
			zap.L().Warn("failed to parse existing ACME user; creating new", zap.Error(err))
		}
	}

	// If we have a usable user (email + key), return it.
	if user.Email != "" && len(user.KeyPEM) > 0 {
		return user
	}

	// New user: generate key and set email.
	privKey, err := certcrypto.GeneratePrivateKey(certcrypto.RSA2048)
	if err != nil {
		zap.L().Fatal("failed to generate ACME private key", zap.Error(err))
	}
	pemBytes := certcrypto.PEMEncode(privKey)

	if email == "" {
		zap.L().Warn("LetsEncryptEmail is empty; ACME account will have empty email")
	}

	return leUser{
		Email:  email,
		KeyPEM: pemBytes,
		key:    privKey,
	}
}
