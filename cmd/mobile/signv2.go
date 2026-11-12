// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"errors"
)

// APK Signature Scheme v2 (https://source.android.com/docs/security/features/apksigning/v2),
// which Android requires for apps that target SDK version 30 or later (in
// addition to the v1 JAR signature of [writer.Close], for older versions
// of Android): an APK Signing Block, between the files of the APK (a zip
// file) and its central directory, with a signature of the digest of the
// rest of the APK.

const (
	// apkSigV2BlockID is the ID of the v2 signature in the signing block.
	apkSigV2BlockID = 0x7109871a

	// apkSigRSAPKCS1SHA256 is the ID of the signature algorithm:
	// RSASSA-PKCS1-v1_5 with SHA2-256.
	apkSigRSAPKCS1SHA256 = 0x0103

	// apkSigChunkSize is the size of the chunks of the digest.
	apkSigChunkSize = 1 << 20

	// apkSigBlockMagic ends the signing block.
	apkSigBlockMagic = "APK Sig Block 42"
)

// signV2 returns the given APK (a zip file, signed with v1) with a v2
// signature of the given key and certificate (DER), in an APK Signing
// Block inserted before its central directory.
func signV2(apk []byte, priv *rsa.PrivateKey, cert []byte) ([]byte, error) {
	eocd := findEOCD(apk)
	if eocd < 0 {
		return nil, errors.New("apk: no end of central directory")
	}
	cdOffset := int(binary.LittleEndian.Uint32(apk[eocd+16:]))
	if cdOffset > eocd {
		return nil, errors.New("apk: invalid central directory offset")
	}
	entries, cd, end := apk[:cdOffset], apk[cdOffset:eocd], apk[eocd:]

	// the digest of the entries, the central directory, and the end of
	// central directory, whose offset of the central directory is that of
	// the signing block, which it is here, before it is inserted
	digest := apkV2Digest(entries, cd, end)

	// signed data: digests, certificates, additional attributes (none)
	signedData := lenPrefixed(concat(
		lenPrefixed(lenPrefixed(concat(u32(apkSigRSAPKCS1SHA256), lenPrefixed(digest)))),
		lenPrefixed(lenPrefixed(cert)),
		lenPrefixed(nil),
	))
	sd := signedData[4:] // (without its length, as it is signed)
	h := sha256.Sum256(sd)
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		return nil, err
	}
	pub, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, err
	}
	signer := lenPrefixed(concat(
		signedData,
		lenPrefixed(lenPrefixed(concat(u32(apkSigRSAPKCS1SHA256), lenPrefixed(sig)))),
		lenPrefixed(pub),
	))
	value := lenPrefixed(signer) // the sequence of signers

	// the signing block: its size, the ID-value pair, its size, and magic
	pair := concat(u64(uint64(4+len(value))), u32(apkSigV2BlockID), value)
	size := uint64(len(pair) + 8 + len(apkSigBlockMagic))
	block := concat(u64(size), pair, u64(size), []byte(apkSigBlockMagic))

	// the end of central directory with the new offset of the central
	// directory, after the signing block
	newEnd := bytes.Clone(end)
	binary.LittleEndian.PutUint32(newEnd[16:], uint32(cdOffset+len(block)))
	return concat(entries, block, cd, newEnd), nil
}

// apkV2Digest returns the SHA-256 digest of the given sections of the APK
// for its v2 signature: of the digests of their chunks.
func apkV2Digest(sections ...[]byte) []byte {
	var chunks [][]byte
	for _, s := range sections {
		for len(s) > 0 {
			n := min(len(s), apkSigChunkSize)
			chunks = append(chunks, s[:n])
			s = s[n:]
		}
	}
	top := sha256.New()
	top.Write([]byte{0x5a})
	top.Write(u32(uint32(len(chunks))))
	for _, c := range chunks {
		h := sha256.New()
		h.Write([]byte{0xa5})
		h.Write(u32(uint32(len(c))))
		h.Write(c)
		top.Write(h.Sum(nil))
	}
	return top.Sum(nil)
}

// findEOCD returns the offset of the end of central directory record of
// the given zip file, or -1 if there is none.
func findEOCD(b []byte) int {
	const eocdLen = 22
	for i := len(b) - eocdLen; i >= 0 && i >= len(b)-eocdLen-0xffff; i-- {
		if binary.LittleEndian.Uint32(b[i:]) == 0x06054b50 &&
			int(binary.LittleEndian.Uint16(b[i+20:])) == len(b)-i-eocdLen {
			return i
		}
	}
	return -1
}

// lenPrefixed returns the given bytes with their length before them
// (a little-endian uint32).
func lenPrefixed(b []byte) []byte {
	return concat(u32(uint32(len(b))), b)
}

func u32(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }

func u64(v uint64) []byte { return binary.LittleEndian.AppendUint64(nil, v) }

func concat(bs ...[]byte) []byte { return bytes.Join(bs, nil) }
