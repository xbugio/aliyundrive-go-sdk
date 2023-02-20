package aliyundrive

import (
	"context"
	"encoding/json"
)

type GetUserInfoRequest struct {
}

type GetUserInfoResponse struct {
	DomainID       string `json:"domain_id"`
	UserID         string `json:"user_id"`
	Avatar         string `json:"avatar"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	Email          string `json:"email"`
	NickName       string `json:"nick_name"`
	Phone          string `json:"phone"`
	PhoneRegion    string `json:"phone_region"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	UserName       string `json:"user_name"`
	Description    string `json:"description"`
	DefaultDriveID string `json:"default_drive_id"`
	UserData       struct {
	} `json:"user_data"`
	DenyChangePasswordBySelf    bool        `json:"deny_change_password_by_self"`
	NeedChangePasswordNextLogin bool        `json:"need_change_password_next_login"`
	Creator                     string      `json:"creator"`
	ExpiredAt                   int         `json:"expired_at"`
	Permission                  interface{} `json:"permission"`
	DefaultLocation             string      `json:"default_location"`
	LastLoginTime               int64       `json:"last_login_time"`
}

// 获取用户信息接口
func (c *Drive) DoGetUserInfoRequest(ctx context.Context, request GetUserInfoRequest) (*GetUserInfoResponse, error) {
	httpRequest, err := c.toRequest(ctx, "https://api.aliyundrive.com/v2/user/get", Object{})
	if err != nil {
		return nil, err
	}
	if err := c.withCredit(ctx, httpRequest); err != nil {
		return nil, err
	}
	resp, err := c.doRequest(httpRequest)
	if err != nil {
		return nil, err
	}

	result := new(GetUserInfoResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// 刷新accesstoken接口，该接口只需要refresh token，不需要accesstoken
func (c *Drive) DoRefreshTokenRequest(ctx context.Context, request RefreshTokenRequest) (*RefreshTokenResponse, error) {
	resp, err := c.requestWithoutCredit(ctx, "https://api.aliyundrive.com/token/refresh", request)
	if err != nil {
		return nil, err
	}

	result := new(RefreshTokenResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type CreateSessionRequest struct {
	DeviceName string `json:"deviceName"`
	ModelName  string `json:"modelName"`
	PubKey     string `json:"pubKey"`
	Signature  string `json:"-"`
}

type CreateSessionResponse struct {
	Result  bool `json:"result"`
	Success bool `json:"success"`
}

// 发布publicKey到服务器，该接口不需要signature，但需要accesstoken
func (c *Drive) DoCreateSessionRequest(ctx context.Context, request CreateSessionRequest) (*CreateSessionResponse, error) {
	httpRequest, err := c.toRequest(ctx, "https://api.aliyundrive.com/users/v1/users/device/create_session", request)
	if err != nil {
		return nil, err
	}
	if err := c.withCredit(ctx, httpRequest); err != nil {
		return nil, err
	}
	httpRequest.Header.Set("X-Signature", request.Signature)
	resp, err := c.doRequest(httpRequest)
	if err != nil {
		return nil, err
	}

	result := new(CreateSessionResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type RenewSessionRequest struct {
}

type RenewSessionResponse struct {
}

// 刷新session
func (c *Drive) DoRenewSessionRequest(ctx context.Context, request RenewSessionRequest) (*RenewSessionResponse, error) {
	httpRequest, err := c.toRequest(ctx, "https://api.aliyundrive.com/users/v1/users/device/renew_session", Object{})
	if err != nil {
		return nil, err
	}
	if err := c.withCredit(ctx, httpRequest); err != nil {
		return nil, err
	}
	resp, err := c.doRequest(httpRequest)
	if err != nil {
		return nil, err
	}

	result := new(RenewSessionResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
