package aliyundrive

import (
	"net/http"
)

// 操作阿里网盘的SDK客户端
type Drive struct {
	driveId          string
	deviceId         string
	tokenManager     TokenManager
	signatureManager SignatureManager
	httpClient       *http.Client
}

type optionFunc func(c *Drive)

// 创建一个新的阿里网盘SDK客户端
//
// options 配置项
func New(options ...optionFunc) *Drive {
	c := new(Drive)
	c.SetOption(options...)
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	return c
}

// 设置配置项
func (c *Drive) SetOption(options ...optionFunc) *Drive {
	for _, setOption := range options {
		setOption(c)
	}
	return c
}

// 配置DriveId
func WithDriveId(driveId string) optionFunc {
	return func(c *Drive) {
		c.driveId = driveId
	}
}

// 配置DeviceId
func WithDeviceId(deviceId string) optionFunc {
	return func(c *Drive) {
		c.deviceId = deviceId
	}
}

// 配置自定义Http客户端
func WithHttpClient(httpClient *http.Client) optionFunc {
	return func(c *Drive) {
		c.httpClient = httpClient
	}
}

// 配置Token管理器
func WithTokenManager(tokenManager TokenManager) optionFunc {
	return func(c *Drive) {
		c.tokenManager = tokenManager
	}
}

// 配置签名管理器
func WithSignatureManager(signatureManager SignatureManager) optionFunc {
	return func(c *Drive) {
		c.signatureManager = signatureManager
	}
}
