package aliyundrive

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dustinxie/ecc"
)

type SignatureManager interface {
	Signature(ctx context.Context) (string, error)
}

type signatureManager struct {
	drive      *Drive
	appId      string
	deviceId   string
	userId     string
	httpClient *http.Client

	privateKey  *ecdsa.PrivateKey
	expiredTime time.Time
	signature   string
}

func NewSignatureManager(drive *Drive, appId string, deviceId string, userId string, httpClient *http.Client) *signatureManager {
	m := &signatureManager{
		drive:    drive,
		appId:    appId,
		deviceId: deviceId,
		userId:   userId,
	}
	if httpClient == nil {
		m.httpClient = http.DefaultClient
	} else {
		m.httpClient = httpClient
	}
	return m
}

func (m *signatureManager) Signature(ctx context.Context) (string, error) {
	if m.expiredTime.Before(time.Now()) {
		err := m.genSignature(ctx)
		if err != nil {
			return "", err
		}
	}
	return m.signature, nil
}

func (m *signatureManager) genSignature(ctx context.Context) error {

	// 生成privateKey
	privateKey, err := ecdsa.GenerateKey(ecc.P256k1(), rand.Reader)
	if err != nil {
		return err
	}

	// 获取publicKey
	publicKeyString := m.getPublicKeyString(privateKey)

	// 用key去生成signature
	// FIXME: 临时采用远端node换算signature
	code := ""
	hasher := sha256.New()
	hasher.Write([]byte(code))
	sum := hasher.Sum(nil)
	msg := base64.StdEncoding.EncodeToString(sum)
	key := base64.StdEncoding.EncodeToString(privateKey.D.Bytes())

	params := make(url.Values)
	params.Set("msg", msg)
	params.Set("key", key)
	api := "https://splatoon3.doubi.fun:8443/aliyundrive-sign?" + params.Encode()
	resp, err := m.httpClient.Get(api)
	if err != nil {
		return err
	}
	respData, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	signatureData, err := base64.StdEncoding.DecodeString(string(respData))
	if err != nil {
		return err
	}
	signature := hex.EncodeToString(signatureData) + "00"

	// 提交key到阿里云
	now := time.Now()
	_, err = m.drive.DoCreateSessionRequest(ctx, CreateSessionRequest{
		DeviceName: "Chrome浏览器",
		ModelName:  "Mac OS网页版",
		PubKey:     publicKeyString,
		Signature:  signature,
	})
	if err != nil {
		return err
	}

	// 没问题则更新存储
	m.signature = signature
	m.privateKey = privateKey
	m.expiredTime = now.Add(time.Second * 82800)
	return nil
}

func (m *signatureManager) getPublicKeyString(privateKey *ecdsa.PrivateKey) string {
	xData := privateKey.PublicKey.X.Bytes()
	yData := privateKey.PublicKey.Y.Bytes()
	data := make([]byte, 65)
	data[0] = 0x04
	copy(data[1+(32-len(xData)):], xData)
	copy(data[33+(32-len(yData)):], yData)
	return hex.EncodeToString(data)
}
