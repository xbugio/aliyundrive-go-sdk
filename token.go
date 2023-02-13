package aliyundrive

import (
	"context"
	"sync"
	"time"
)

// Token管理器
type TokenManager interface {
	AccessToken(ctx context.Context) (string, error)
}

type staticTokenManager struct {
	accessToken string
}

// 创建一个静态Token管理器
//
// 静态Token管理器每次调用AccessToken返回的都是固定的token，
// 一般建议用于另外有token管理的机制，而程序只运行使用token很少次的场景，
// 免于频繁刷新token
func NewStaticTokenManager(accessToken string) *staticTokenManager {
	return &staticTokenManager{accessToken: accessToken}
}

func (m *staticTokenManager) AccessToken(ctx context.Context) (string, error) {
	return m.accessToken, nil
}

type refreshTokenManager struct {
	drive                 *Drive
	refreshToken          string
	accessToken           string
	accessTokenExpireTime time.Time
	lock                  *sync.Mutex
}

// 创建一个RefreshToken管理器
//
// RefreshToken管理器，通过RefreshToken，利用refresh接口来获取有效的accesstoken，
// 内部会记录accesstoken的有效期，若accesstoken已经失效会重新refresh。
//
// 注意，该管理器不会在accesstoken失效后立马自动refresh，而是在下次调用获取时判断是否refresh。
func NewRefreshTokenManager(drive *Drive, refreshToken string) *refreshTokenManager {
	return &refreshTokenManager{
		drive:                 drive,
		refreshToken:          refreshToken,
		accessTokenExpireTime: time.Unix(0, 0),
		lock:                  new(sync.Mutex),
	}
}

func (m *refreshTokenManager) AccessToken(ctx context.Context) (string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	now := time.Now()
	if now.Before(m.accessTokenExpireTime) {
		return m.accessToken, nil
	}

	err := m.refresh(ctx)
	if err != nil {
		return "", err
	}
	return m.accessToken, nil
}

func (m *refreshTokenManager) refresh(ctx context.Context) error {
	now := time.Now()
	resp, err := m.drive.DoRefreshTokenRequest(ctx, RefreshTokenRequest{
		RefreshToken: m.refreshToken,
	})
	if err != nil {
		return err
	}
	m.refreshToken = resp.RefreshToken
	m.accessToken = resp.AccessToken
	m.accessTokenExpireTime = now.Add(time.Second * time.Duration(resp.ExpiresIn-60))
	return nil
}

type keepAliveTokenManager struct {
	tokenManager TokenManager
	wg           *sync.WaitGroup
}

// 创建一个保活Token管理器
//
// 保活Token管理器会在调用KeepAlive后按照设定的间隔定时去请求accesstoken，
// 搭配RefreshTokenManager使用可实现refreshtoken和accesstoken的保活不失效
//
// tokenManager：实际需要保活的token管理器
func NewKeepAliveTokenManager(tokenManager TokenManager) *keepAliveTokenManager {
	return &keepAliveTokenManager{
		tokenManager: tokenManager,
		wg:           new(sync.WaitGroup),
	}
}

func (m *keepAliveTokenManager) AccessToken(ctx context.Context) (string, error) {
	return m.tokenManager.AccessToken(ctx)
}

// 开始保活
//
// ctx：当ctx Done事件到来之后，则结束保活
//
// t：保活时间间隔
func (m *keepAliveTokenManager) KeepAlive(ctx context.Context, t time.Duration) {
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
			m.AccessToken(ctx)
		}
		ticker.Stop()
	}()
}

// 当KeepAlive的ctx Done事件到来，调用WaitStop，等待保活任务完全终止
func (m *keepAliveTokenManager) WaitStop() {
	m.wg.Wait()
}
