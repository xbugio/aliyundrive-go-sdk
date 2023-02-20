package aliyundrive

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/dustinxie/ecc"
)

type SignatureManager interface {
	Signature(ctx context.Context) (string, error)
}

type signatureManager struct {
	drive *Drive

	privateKey  *ecdsa.PrivateKey
	expiredTime time.Time
	signature   string
	wg          *sync.WaitGroup
}

func NewSignatureManager(drive *Drive) *signatureManager {
	m := &signatureManager{
		drive: drive,
		wg:    new(sync.WaitGroup),
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

func (m *signatureManager) KeepAlive(ctx context.Context, t time.Duration) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(t)
	keepaliveLoop:
		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				break keepaliveLoop
			}
			m.Signature(ctx)
		}
		ticker.Stop()
	}()
}

func (m *signatureManager) WaitStop() {
	m.wg.Wait()
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
	code := "5dde4e1bdf9e4966b387ba58f4b3fdc3:" + m.drive.deviceId + ":" + m.drive.userId + ":0"
	hasher := sha256.New()
	hasher.Write([]byte(code))
	sum := hasher.Sum(nil)

	signatureData, err := ecc.SignBytes(privateKey, sum, ecc.RecID|ecc.LowerS)
	if err != nil {
		return err
	}
	signature := hex.EncodeToString(signatureData)

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
	m.expiredTime = now.Add(time.Minute * 3)
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
