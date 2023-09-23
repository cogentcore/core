// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"io"
	"math/big"
	"time"
)

// SignPKCS7 does the minimal amount of work necessary to embed an RSA
// signature into a PKCS#7 certificate.
//
// We prepare the certificate using the x509 package, read it back in
// to our custom data type and then write it back out with the signature.
func SignPKCS7(rand io.Reader, priv *rsa.PrivateKey, msg []byte) ([]byte, error) {
	const serialNumber = 0x5462c4dd // arbitrary
	name := pkix.Name{CommonName: "gomobile"}

	template := &x509.Certificate{
		SerialNumber:       big.NewInt(serialNumber),
		SignatureAlgorithm: x509.SHA1WithRSA,
		Subject:            name,
	}

	b, err := x509.CreateCertificate(rand, template, template, priv.Public(), priv)
	if err != nil {
		return nil, err
	}

	c := Certificate{}
	if _, err := asn1.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	h := sha1.New()
	h.Write(msg)
	hashed := h.Sum(nil)

	signed, err := rsa.SignPKCS1v15(rand, priv, crypto.SHA1, hashed)
	if err != nil {
		return nil, err
	}

	content := Pkcs7SignedData{
		ContentType: OIDSignedData,
		Content: SignedData{
			Version: 1,
			DigestAlgorithms: []pkix.AlgorithmIdentifier{{
				Algorithm:  OIDSHA1,
				Parameters: asn1.RawValue{Tag: 5},
			}},
			ContentInfo:  ContentInfo{Type: OIDData},
			Certificates: c,
			SignerInfos: []SignerInfo{{
				Version: 1,
				IssuerAndSerialNumber: IssuerAndSerialNumber{
					Issuer:       name.ToRDNSequence(),
					SerialNumber: serialNumber,
				},
				DigestAlgorithm: pkix.AlgorithmIdentifier{
					Algorithm:  OIDSHA1,
					Parameters: asn1.RawValue{Tag: 5},
				},
				DigestEncryptionAlgorithm: pkix.AlgorithmIdentifier{
					Algorithm:  OIDRSAEncryption,
					Parameters: asn1.RawValue{Tag: 5},
				},
				EncryptedDigest: signed,
			}},
		},
	}

	return asn1.Marshal(content)
}

type Pkcs7SignedData struct {
	ContentType asn1.ObjectIdentifier
	Content     SignedData `asn1:"tag:0,explicit"`
}

// SignedData is defined in rfc2315, section 9.1.
type SignedData struct {
	Version          int
	DigestAlgorithms []pkix.AlgorithmIdentifier `asn1:"set"`
	ContentInfo      ContentInfo
	Certificates     Certificate  `asn1:"tag0,explicit"`
	SignerInfos      []SignerInfo `asn1:"set"`
}

type ContentInfo struct {
	Type asn1.ObjectIdentifier
	// Content is optional in PKCS#7 and not provided here.
}

// Certificate is defined in rfc2459, section 4.1.
type Certificate struct {
	TBSCertificate     TBSCertificate
	SignatureAlgorithm pkix.AlgorithmIdentifier
	SignatureValue     asn1.BitString
}

// TBSCertificate is defined in rfc2459, section 4.1.
type TBSCertificate struct {
	Version      int `asn1:"tag:0,default:2,explicit"`
	SerialNumber int
	Signature    pkix.AlgorithmIdentifier
	Issuer       pkix.RDNSequence // pkix.Name
	Validity     Validity
	Subject      pkix.RDNSequence // pkix.Name
	SubjectPKI   SubjectPublicKeyInfo
}

// Validity is defined in rfc2459, section 4.1.
type Validity struct {
	NotBefore time.Time
	NotAfter  time.Time
}

// SubjectPublicKeyInfo is defined in rfc2459, section 4.1.
type SubjectPublicKeyInfo struct {
	Algorithm        pkix.AlgorithmIdentifier
	SubjectPublicKey asn1.BitString
}

type SignerInfo struct {
	Version                   int
	IssuerAndSerialNumber     IssuerAndSerialNumber
	DigestAlgorithm           pkix.AlgorithmIdentifier
	DigestEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedDigest           []byte
}

type IssuerAndSerialNumber struct {
	Issuer       pkix.RDNSequence // pkix.Name
	SerialNumber int
}

// Various ASN.1 Object Identifies, mostly from rfc3852.
var (
	OIDPKCS7         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7}
	OIDData          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	OIDSignedData    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	OIDSHA1          = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	OIDRSAEncryption = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
)
