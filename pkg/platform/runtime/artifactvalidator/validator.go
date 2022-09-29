package artifactvalidator

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"go.mozilla.org/pkcs7"
)

type signature struct {
	Sig  string `json: "sig"`
	Cert string `json: "cert"`
}

type attestation struct {
	Payload    string      `json: "payload"`
	Signatures []signature `json: "signatures"`
}

func ValidateAttestation(attestationFile string) error {
	data, err := fileutils.ReadFile(attestationFile)
	if err != nil {
		return errs.Wrap(err, "Could not read attestation: "+attestationFile)
	}

	att := attestation{}
	err = json.Unmarshal(data, &att)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshal attestation")
	}

	if len(att.Signatures) == 0 {
		return locale.NewError("validate_attestation_fail_no_signatures", "No signatures")
	}

	// Verify signing certificate.
	pemBlock, _ := pem.Decode([]byte(att.Signatures[0].Cert))
	if pemBlock == nil {
		return errs.Wrap(err, "Unable to decode attestation certificate")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return errs.Wrap(err, "Unable to parse attestation certificate")
	}

	intermediates := x509.NewCertPool()
	addIntermediatesToPool(cert, intermediates)

	opts := x509.VerifyOptions{
		Roots:         nil, // use system root CAs
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}
	_, err = cert.Verify(opts)
	if err != nil {
		return errs.Wrap(err, "Unable to validate certificate")
	}

	// Verify signature.
	payload := make([]byte, len(att.Payload))
	n, err := base64.StdEncoding.Decode(payload, []byte(att.Payload))
	if err != nil {
		return errs.Wrap(err, "Unable to decode attestation payload")
	}
	payload = payload[:n]
	hash := sha256.New()
	hash.Write(payload)
	digest := hash.Sum(nil)

	signature := make([]byte, len(att.Signatures[0].Sig))
	n, err = base64.StdEncoding.Decode(signature, []byte(att.Signatures[0].Sig))
	if err != nil {
		return errs.Wrap(err, "Unable to decode attestation signature")
	}
	signature = signature[:n]

	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return locale.NewError("validate_attestation_fail_public_key", "Certificate's public key is not an expected RSA pubkey")
	}
	err = rsa.VerifyPSS(publicKey, crypto.SHA256, digest, signature, &rsa.PSSOptions{Hash: crypto.SHA256})
	if err != nil {
		return errs.Wrap(err, "Unable to validate signature")
	}

	// TODO: read payload artifact SHAs and validate them against downloaded artifact SHAs.

	return nil
}

func addIntermediatesToPool(cert *x509.Certificate, pool *x509.CertPool) {
	for _, url := range cert.IssuingCertificateURL {
		bytes, err := download.GetDirect(url)
		if err != nil {
			logging.Debug("Unable to download intermediate certificate %s: %v", url, err)
			continue
		}
		if !strings.HasSuffix(url, ".p7c") {
			cert, err := x509.ParseCertificate(bytes)
			if err != nil {
				logging.Debug("Unable to parse intermediate certificate %s: %v", url, err)
				continue
			}
			pool.AddCert(cert)
			addIntermediatesToPool(cert, pool)
		} else {
			p7, err := pkcs7.Parse(bytes)
			if err != nil {
				logging.Debug("Unable to parse intermediate certificate %s: %v", url, err)
				continue
			}
			for _, cert := range p7.Certificates {
				pool.AddCert(cert)
				addIntermediatesToPool(cert, pool)
			}
		}
	}
}

func ValidateChecksum(archivePath string, expectedChecksum string) error {
	if expectedChecksum != "" {
		logging.Debug("Validating checksum for %s", archivePath)
	} else {
		logging.Debug("Skipping checksum validation for %s because the Platform did not provide a checksum to validate against.")
		return nil
	}

	checksum, err := fileutils.Sha256Hash(archivePath)
	if err != nil {
		return errs.Wrap(err, "Failed to compute checksum for "+archivePath)
	}

	if checksum != expectedChecksum {
		logging.Debug("Checksum validation failed. Expected '%s', but was '%s'", expectedChecksum, checksum)
		// Note: the artifact name will be reported higher up the chain
		return locale.WrapError(err, "artifact_checksum_failed", "Checksum validation failed")
	}

	return nil
}
