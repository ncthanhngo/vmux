package chromeimport

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"errors"
	"hash"
)

// macOS Chrome cookie encryption parameters (stable across versions).
const (
	cookieSalt       = "saltysalt"
	cookieIterations = 1003
	cookieKeyLen     = 16
)

// cookieIV is 16 spaces, per Chrome's macOS cookie scheme.
var cookieIV = []byte("                ")

// DeriveKey turns the Keychain "Chrome Safe Storage" password into the AES key
// Chrome uses for cookie values (PBKDF2-HMAC-SHA1, 1003 iters, salt "saltysalt").
func DeriveKey(safeStoragePassword string) []byte {
	return pbkdf2SHA1([]byte(safeStoragePassword), []byte(cookieSalt), cookieIterations, cookieKeyLen)
}

// DecryptCookie decrypts a Chrome `encrypted_value` (v10 prefix, AES-128-CBC)
// using a key from DeriveKey. Returns the plaintext cookie value.
func DecryptCookie(key, encrypted []byte) ([]byte, error) {
	if len(encrypted) < 3 || string(encrypted[:3]) != "v10" {
		return nil, errors.New("chromeimport: not a v10-encrypted cookie")
	}
	ciphertext := encrypted[3:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("chromeimport: bad ciphertext length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, cookieIV).CryptBlocks(plain, ciphertext)
	return unpadPKCS7(plain, aes.BlockSize)
}

// EncryptCookie is the inverse of DecryptCookie (used in tests / write-back).
func EncryptCookie(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := padPKCS7(plaintext, aes.BlockSize)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, cookieIV).CryptBlocks(out, padded)
	return append([]byte("v10"), out...), nil
}

func unpadPKCS7(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("chromeimport: invalid padding")
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, errors.New("chromeimport: invalid padding")
	}
	return data[:len(data)-pad], nil
}

func padPKCS7(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}

// pbkdf2SHA1 implements PBKDF2-HMAC-SHA1 (avoids an x/crypto dependency).
func pbkdf2SHA1(password, salt []byte, iter, keyLen int) []byte {
	prf := func() hash.Hash { return hmac.New(sha1.New, password) }
	hLen := sha1.Size
	numBlocks := (keyLen + hLen - 1) / hLen
	var dk []byte
	for block := 1; block <= numBlocks; block++ {
		h := prf()
		h.Write(salt)
		h.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := h.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for n := 2; n <= iter; n++ {
			h = prf()
			h.Write(u)
			u = h.Sum(nil)
			for i := range t {
				t[i] ^= u[i]
			}
		}
		dk = append(dk, t...)
	}
	return dk[:keyLen]
}
