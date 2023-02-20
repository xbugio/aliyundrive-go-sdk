package aliyundrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Array []any
type Object map[string]any

// 阿里云盘API调用出错的错误信息
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// 错误信息的描述
func (r *ErrorResponse) Error() string {
	return fmt.Sprintf(`{"code":"%v","message":"%v"}`, r.Code, r.Message)
}

// 判断某个error是否是云盘接口返回的PreHashMatched错误
//
// 由于PreHashMatched用于秒传的情况，
// 所以遇到该错误需要走秒传后续的逻辑
func IsPreHashMatchedError(err error) bool {
	errResponse, ok := err.(*ErrorResponse)
	if !ok {
		return false
	}
	return errResponse.Code == "PreHashMatched"
}

func (c *Drive) toRequest(ctx context.Context, url string, params any) (*http.Request, error) {
	bodyData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(bodyData)
	request, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

func (c *Drive) withCredit(ctx context.Context, request *http.Request) error {
	accessToken, err := c.tokenManager.AccessToken(ctx)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("X-Device-Id", c.deviceId)
	return nil
}

func (c *Drive) withSignature(ctx context.Context, request *http.Request) error {
	signature, err := c.signatureManager.Signature(ctx)
	if err != nil {
		return err
	}
	request.Header.Set("X-Signature", signature)
	return nil
}

func (c *Drive) requestWithCredit(ctx context.Context, url string, params any) ([]byte, error) {
	request, err := c.toRequest(ctx, url, params)
	if err != nil {
		return nil, err
	}
	if err := c.withCredit(ctx, request); err != nil {
		return nil, err
	}
	if err := c.withSignature(ctx, request); err != nil {
		return nil, err
	}
	return c.doRequest(request)
}

func (c *Drive) requestWithoutCredit(ctx context.Context, url string, params any) ([]byte, error) {
	request, err := c.toRequest(ctx, url, params)
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(request)
	return resp, err
}

func (c *Drive) doRequest(request *http.Request) ([]byte, error) {
	resp, err := c.HttpClient.Do(request)
	if err != nil {
		return nil, err
	}
	respData, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	result := new(ErrorResponse)
	err = json.Unmarshal(respData, result)
	if err != nil {
		return nil, err
	}

	if result.Code != "" || result.Message != "" {
		return nil, result
	}

	return respData, nil
}
