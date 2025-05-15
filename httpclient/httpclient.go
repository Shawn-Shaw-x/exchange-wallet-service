package httpclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	gresty "github.com/go-resty/resty/v2"
)

var errBlockChainHTTPError = errors.New("blockchain http error")

type NotifyClient struct {
	client *gresty.Client
}

/*新建 http 通知客户端*/
func NewNotifyClient(baseUrl string) (*NotifyClient, error) {
	if baseUrl == "" {
		return nil, errors.New("base url is required")
	}
	client := gresty.New()
	client.SetBaseURL(baseUrl)
	client.OnAfterResponse(func(c *gresty.Client, resp *gresty.Response) error {
		statusCode := resp.StatusCode()
		/*出错响应*/
		if statusCode >= 400 {
			method := resp.Request.Method
			url := resp.Request.URL
			return fmt.Errorf("%s %s %s", method, url, resp.Status())
		}
		return nil
	})

	return &NotifyClient{
		client: client,
	}, nil
}

/*通知方法封装*/
func (nc *NotifyClient) BusinessNotify(notifyData *NotifyRequest) (bool, error) {
	body, err := json.Marshal(notifyData)
	if err != nil {
		log.Error("fail to marshal notifyRequest data", "err", err)
		return false, err
	}

	res, err := nc.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		SetResult(&NotifyResponse{}).Post("/exchange-wallet/notify")
	if err != nil {
		log.Error("fail to send notifyRequest", "err", err)
		return false, err
	}
	spt, ok := res.Result().(*NotifyResponse)
	if !ok {
		return false, errors.New("response is not a NotifyResponse")
	}
	return spt.Success, nil
}
