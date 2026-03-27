package weixin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

const (
	DefaultBaseURL     = "https://ilinkai.weixin.qq.com"
	BotType            = "3"
	ChannelVersion     = "1.0.2"
	MessageTypeUser    = 1
	MessageTypeBot     = 2
	MessageStateFinish = 2
	ItemTypeText       = 1
	ItemTypeVoice      = 3
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) GetQRCode() (*QRCodeResponse, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/ilink/bot/get_bot_qrcode?bot_type=%s", c.baseURL, BotType), nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result QRCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) PollQRCodeStatus(ctx context.Context, qrcode string, timeout time.Duration) (*QRCodeStatusResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("%s/ilink/bot/get_qrcode_status?qrcode=%s", c.baseURL, qrcode)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		c.setHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			time.Sleep(2 * time.Second)
			continue
		}

		var result QRCodeStatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			time.Sleep(2 * time.Second)
			continue
		}
		resp.Body.Close()

		if result.Status == "confirmed" {
			return &result, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return nil, fmt.Errorf("QR code login timed out after %v", timeout)
}

func (c *Client) GetUpdates(ctx context.Context, buf string) (*GetUpdatesResponse, error) {
	body := GetUpdatesRequest{
		GetUpdatesBuf: buf,
		BaseInfo:      baseInfo(),
	}
	var result GetUpdatesResponse
	if err := c.post(ctx, "/ilink/bot/getupdates", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SendTextMessage(ctx context.Context, toUserID, text, contextToken string) error {
	body := SendMessageRequest{
		Msg: WeixinMessage{
			ToUserID:     toUserID,
			ClientID:     generateClientID(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ContextToken: contextToken,
			ItemList: []MessageItem{
				{Type: ItemTypeText, TextItem: &TextItem{Text: text}},
			},
		},
		BaseInfo: baseInfo(),
	}

	var result SendMessageResponse
	if err := c.post(ctx, "/ilink/bot/sendmessage", body, &result); err != nil {
		return err
	}
	if result.Ret != 0 {
		return fmt.Errorf("sendmessage ret=%d errcode=%d: %s", result.Ret, result.ErrCode, result.Message)
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iLink API %s returned %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	uin := strconv.FormatUint(uint64(rand.Uint32()), 10)
	req.Header.Set("X-WECHAT-UIN", base64.StdEncoding.EncodeToString([]byte(uin)))
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func baseInfo() BaseInfo {
	return BaseInfo{ChannelVersion: ChannelVersion}
}

func generateClientID() string {
	return fmt.Sprintf("openclaw-weixin-%d-%d", time.Now().UnixMilli(), rand.IntN(100000))
}
