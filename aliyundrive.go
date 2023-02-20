package aliyundrive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

// 操作阿里网盘的SDK客户端
type Drive struct {
	RefreshToken string
	HttpClient   *http.Client

	ctx    context.Context
	cancel context.CancelFunc

	tokenManager     *keepAliveTokenManager
	signatureManager *signatureManager

	driveId  string
	userId   string
	deviceId string
}

// 初始化参数
func (c *Drive) Init() error {
	if c.HttpClient == nil {
		c.HttpClient = http.DefaultClient
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.tokenManager = NewKeepAliveTokenManager(NewRefreshTokenManager(c, c.RefreshToken))
	c.tokenManager.KeepAlive(c.ctx, time.Second*10)

	resp, err := c.DoGetUserInfoRequest(c.ctx, GetUserInfoRequest{})
	if err != nil {
		c.Destory()
		return err
	}

	c.driveId = resp.DefaultDriveID
	c.userId = resp.UserID
	hasher := sha256.New()
	hasher.Write([]byte(c.userId))
	c.deviceId = hex.EncodeToString(hasher.Sum(nil))

	c.signatureManager = NewSignatureManager(c)
	c.signatureManager.KeepAlive(c.ctx, time.Second*10)

	return nil
}

func (c *Drive) Destory() {
	c.cancel()
	c.tokenManager.WaitStop()
	c.signatureManager.WaitStop()
}
