package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/youmark/pkcs8"
)

func LoadX509KeyPairWithPassword(certFile, keyFile string, password []byte) (tls.Certificate, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("read cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("read key: %w", err)
	}

	// Разбираем цепочку сертификатов
	var certs [][]byte
	for {
		var b *pem.Block
		b, certPEM = pem.Decode(certPEM)
		if b == nil {
			break
		}
		if b.Type == "CERTIFICATE" {
			certs = append(certs, b.Bytes)
		}
	}
	if len(certs) == 0 {
		return tls.Certificate{}, errors.New("no certificate found in certFile")
	}

	// Разбираем приватный ключ (включая расшифровку при необходимости)
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return tls.Certificate{}, errors.New("no key found in keyFile")
	}

	var privKey any

	switch block.Type {
	case "ENCRYPTED PRIVATE KEY":
		// Encrypted PKCS#8
		k, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, password)
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("parse encrypted PKCS#8: %w", err)
		}
		privKey = k

	default:
		// Незашифрованный PEM
		der := block.Bytes

		var err error
		switch block.Type {
		case "RSA PRIVATE KEY":
			privKey, err = x509.ParsePKCS1PrivateKey(der)
		case "EC PRIVATE KEY":
			privKey, err = x509.ParseECPrivateKey(der)
		case "PRIVATE KEY":
			// Не зашифрованный PKCS#8
			privKey, err = x509.ParsePKCS8PrivateKey(der)
		default:
			err = fmt.Errorf("unsupported key type: %s", block.Type)
		}
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("parse private key: %w", err)
		}
	}

	cert := tls.Certificate{
		Certificate: certs,
		PrivateKey:  privKey,
	}
	if leaf, err := x509.ParseCertificate(certs[0]); err == nil {
		cert.Leaf = leaf
	}
	return cert, nil
}
