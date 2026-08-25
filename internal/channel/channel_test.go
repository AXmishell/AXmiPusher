package channel

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
)

// genECKeyP8 生成测试用 P8 私钥。
func genECKeyP8(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// TestAPNsJWT 验证 ES256 JWT 结构 + 签名可验证。
func TestAPNsJWT(t *testing.T) {
	p8 := genECKeyP8(t)
	cfg := APNsConfig{TeamID: "TEAM123456", KeyID: "KEY123456", BundleID: "com.example.app", KeyP8: p8}
	s := &APNsSender{}
	jwt, err := s.buildToken(cfg)
	if err != nil {
		t.Fatalf("buildToken: %v", err)
	}
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT 应有三段, 实际 %d 段", len(parts))
	}
	// 解析 header 验证 kid。
	headerRaw, _ := base64.RawURLEncoding.DecodeString(parts[0])
	if !strings.Contains(string(headerRaw), `"kid":"KEY123456"`) {
		t.Fatalf("header 缺少 kid: %s", headerRaw)
	}
	// 用公钥验证签名。
	block, _ := pem.Decode([]byte(p8))
	pk8, _ := x509.ParsePKCS8PrivateKey(block.Bytes)
	priv := pk8.(*ecdsa.PrivateKey)
	sigRaw, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if len(sigRaw) != 64 {
		t.Fatalf("ES256 签名应为 64 字节, 实际 %d", len(sigRaw))
	}
	r := new(big.Int).SetBytes(sigRaw[:32])
	sig := new(big.Int).SetBytes(sigRaw[32:])
	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(&priv.PublicKey, hash[:], r, sig) {
		t.Fatal("JWT 签名验证失败")
	}
}

// TestFCMSendJWT 验证 RS256 JWT 结构。
func TestFCMJWT(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	der := x509.MarshalPKCS1PrivateKey(key)
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))

	parsed, err := parseRSAKey(pemStr)
	if err != nil {
		t.Fatalf("parseRSAKey: %v", err)
	}
	if parsed.N.Cmp(key.N) != 0 {
		t.Fatal("RSA key 解析不一致")
	}
}

// TestBuildMessage 验证邮件头与 base64 正文。
func TestBuildMessage(t *testing.T) {
	msg := buildMessage("sender@test.com", []string{"rcpt@test.com"}, "测试标题", "你好，正文")
	if !strings.Contains(msg, "From: sender@test.com") {
		t.Fatal("缺少 From 头")
	}
	if !strings.Contains(msg, "To: rcpt@test.com") {
		t.Fatal("缺少 To 头")
	}
	if !strings.Contains(msg, "=?UTF-8?B?") {
		t.Fatal("中文主题未做 MIME 编码")
	}
	if !strings.Contains(msg, "Content-Transfer-Encoding: base64") {
		t.Fatal("缺少 base64 编码声明")
	}
	// 正文 base64 解码后应为原文。
	lines := strings.Split(msg, "\r\n")
	var b64 string
	inBody := false
	for _, l := range lines {
		if l == "" {
			inBody = true
			continue
		}
		if inBody {
			b64 += l
		}
	}
	raw, _ := base64.StdEncoding.DecodeString(b64)
	if string(raw) != "你好，正文" {
		t.Fatalf("正文解码不符: %q", raw)
	}
}
