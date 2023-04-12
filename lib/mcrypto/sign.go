package mcrypto

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	headerSigKey      = "X-Sig"
	headerSignTimeKey = "X-Sign-Time"
)

// Signer 签名器
// 用于对 http.Request 进行签名和验证
// 签名时，会增加 X-Sig 和 X-Sign-Time 两个 Header
// 验证时，会从 X-Sig 中获取签名，并与计算出的签名进行比较，同时会验证 X-Sign-Time 是否在 1 分钟内
type Signer interface {
	SignRequest(r *http.Request) error
	VerifyRequest(r *http.Request) error
}

// signer 签名器实现
type signer struct {
	hmacKey    string
	headerKeys []string
}

func NewSigner(hmacKey string, headerKeysToSign ...string) Signer {
	return &signer{hmacKey: hmacKey, headerKeys: headerKeysToSign}
}

func (s *signer) SignRequest(r *http.Request) error {
	return signRequest(r, s.hmacKey, s.headerKeys...)
}

func (s *signer) VerifyRequest(r *http.Request) error {
	return verifyRequest(r, s.hmacKey, s.headerKeys...)
}

// verifyRequest 验证 http.Request 的签名
func verifyRequest(r *http.Request, hmacKey string, headerKeys ...string) error {
	// 利用 HMAC 签名验证请求
	// 1. 从请求头中获取签名
	sig := r.Header.Get(headerSigKey)
	if sig == "" {
		return errors.New("no signature")
	}

	expectedSig, err := calcSignature(hmacKey, r, headerKeys...)
	if err != nil {
		return err
	}

	// 2. 比较签名
	if sig != expectedSig {
		return fmt.Errorf("signature mismatch: %s != %s", sig, expectedSig)
	}

	// 3. 验证时间戳
	t := r.Header.Get(headerSignTimeKey)
	if t == "" {
		return errors.New("no timestamp")
	}
	tm, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return err
	}
	if time.Since(tm) > 1*time.Minute {
		return errors.New("timestamp expired")
	}

	return nil
}

// signRequest 签名 http.Request
func signRequest(r *http.Request, hmacKey string, headerKeys ...string) error {
	// 为 Header 增加一个时间戳
	// 获取当前的时间信息，采用 ISO8601 格式
	t := time.Now().UTC().Format(time.RFC3339)
	r.Header.Add(headerSignTimeKey, t)

	// 计算签名
	sig, err := calcSignature(hmacKey, r, headerKeys...)
	if err != nil {
		return err
	}
	r.Header.Set(headerSigKey, sig)
	return nil
}

// 计算 HMAC 签名并返回
// 1. 从请求体中读取数据, 最大 1MB
// 2. 重新设置请求体
// 3. 计算签名
// 4. 返回签名
func calcSignature(key string, r *http.Request, headerKeysToSign ...string) (string, error) {
	// 1. 计算签名
	mac := hmac.New(sha256.New, []byte(key))

	// 增加 URL 的签名
	mac.Write([]byte(r.URL.String()))
	// 增加 Method 的签名
	mac.Write([]byte(r.Method))
	// 增加 Header 的签名
	for k, v := range r.Header {
		// 排除掉 X-Signature
		if k == headerSigKey {
			continue
		}

		// 仅对 headerKeysToSign 中的 Header 增加签名
		for _, k2 := range headerKeysToSign {
			if k == k2 {
				// log.Printf("k: %s, v: %s\n", k, v[0])
				mac.Write([]byte(k))
				mac.Write([]byte(v[0]))
			}
		}
	}

	if r.Body != nil {
		// 从请求体中读取数据, 最大 1MB
		r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}

		// 重新设置请求体
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		// 增加 Body 的签名
		mac.Write(body)
	}

	expectedMAC := mac.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(expectedMAC)

	// 2. 返回签名
	return sig, nil
}
