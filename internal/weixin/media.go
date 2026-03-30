package weixin

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/md5"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

const (
	DefaultCDNBaseURL   = "https://novac2c.cdn.weixin.qq.com/c2c"
	ItemTypeFile        = 4
	UploadMediaTypeFile = 3
	fileEncryptType     = 1
)

type uploadedFile struct {
	FileName                    string
	FileSize                    int64
	DownloadEncryptedQueryParam string
	AESKey                      []byte
}

func (c *Client) GetUploadURL(ctx context.Context, req GetUploadURLRequest) (*GetUploadURLResponse, error) {
	req.BaseInfo = baseInfo()

	var result GetUploadURLResponse
	if err := c.post(ctx, "/ilink/bot/getuploadurl", req, &result); err != nil {
		return nil, err
	}
	if result.Ret != 0 {
		return nil, fmt.Errorf("getuploadurl ret=%d errcode=%d: %s", result.Ret, result.ErrCode, result.Message)
	}
	return &result, nil
}

func (c *Client) SendFileMessage(ctx context.Context, toUserID, contextToken, filePath string) error {
	uploaded, err := c.uploadFileAttachment(ctx, toUserID, filePath)
	if err != nil {
		return err
	}

	body := SendMessageRequest{
		Msg: WeixinMessage{
			ToUserID:     toUserID,
			ClientID:     generateClientID(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ContextToken: contextToken,
			ItemList: []MessageItem{
				{
					Type: ItemTypeFile,
					FileItem: &FileItem{
						Media: &CDNMedia{
							EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
							AESKey:            base64.StdEncoding.EncodeToString(uploaded.AESKey),
							EncryptType:       fileEncryptType,
						},
						FileName: uploaded.FileName,
						Len:      fmt.Sprintf("%d", uploaded.FileSize),
					},
				},
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

func (c *Client) uploadFileAttachment(ctx context.Context, toUserID, filePath string) (uploadedFile, error) {
	plaintext, err := os.ReadFile(filePath)
	if err != nil {
		return uploadedFile{}, err
	}

	aesKey := make([]byte, 16)
	if _, err := crand.Read(aesKey); err != nil {
		return uploadedFile{}, fmt.Errorf("generate aes key: %w", err)
	}
	fileKey := make([]byte, 16)
	if _, err := crand.Read(fileKey); err != nil {
		return uploadedFile{}, fmt.Errorf("generate file key: %w", err)
	}

	ciphertext, err := encryptAESECB(plaintext, aesKey)
	if err != nil {
		return uploadedFile{}, err
	}

	sum := md5.Sum(plaintext)
	uploadURL, err := c.GetUploadURL(ctx, GetUploadURLRequest{
		FileKey:     hex.EncodeToString(fileKey),
		MediaType:   UploadMediaTypeFile,
		ToUserID:    toUserID,
		RawSize:     int64(len(plaintext)),
		RawFileMD5:  hex.EncodeToString(sum[:]),
		FileSize:    int64(len(ciphertext)),
		NoNeedThumb: true,
		AESKey:      hex.EncodeToString(aesKey),
	})
	if err != nil {
		return uploadedFile{}, err
	}

	targetURL := strings.TrimSpace(uploadURL.UploadFullURL)
	if targetURL == "" {
		if strings.TrimSpace(uploadURL.UploadParam) == "" {
			return uploadedFile{}, fmt.Errorf("getuploadurl returned no upload target")
		}
		targetURL = buildCDNUploadURL(DefaultCDNBaseURL, uploadURL.UploadParam, hex.EncodeToString(fileKey))
	}

	downloadParam, err := c.uploadCiphertext(ctx, targetURL, ciphertext)
	if err != nil {
		return uploadedFile{}, err
	}

	return uploadedFile{
		FileName:                    fileBaseName(filePath),
		FileSize:                    int64(len(plaintext)),
		DownloadEncryptedQueryParam: downloadParam,
		AESKey:                      aesKey,
	}, nil
}

func (c *Client) uploadCiphertext(ctx context.Context, targetURL string, ciphertext []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(ciphertext))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cdn upload returned %d", resp.StatusCode)
	}

	downloadParam := strings.TrimSpace(resp.Header.Get("x-encrypted-param"))
	if downloadParam == "" {
		return "", fmt.Errorf("cdn upload response missing x-encrypted-param header")
	}
	return downloadParam, nil
}

func encryptAESECB(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	padding := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	if padding == 0 {
		padding = aes.BlockSize
	}

	padded := make([]byte, len(plaintext)+padding)
	copy(padded, plaintext)
	for idx := len(plaintext); idx < len(padded); idx++ {
		padded[idx] = byte(padding)
	}

	ciphertext := make([]byte, len(padded))
	for start := 0; start < len(padded); start += aes.BlockSize {
		block.Encrypt(ciphertext[start:start+aes.BlockSize], padded[start:start+aes.BlockSize])
	}
	return ciphertext, nil
}

func buildCDNUploadURL(cdnBaseURL, uploadParam, fileKey string) string {
	return fmt.Sprintf("%s/upload?encrypted_query_param=%s&filekey=%s",
		strings.TrimRight(strings.TrimSpace(cdnBaseURL), "/"),
		url.QueryEscape(strings.TrimSpace(uploadParam)),
		url.QueryEscape(strings.TrimSpace(fileKey)),
	)
}

func fileBaseName(filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "file"
	}

	base := path.Base(strings.ReplaceAll(filePath, "\\", "/"))
	if base == "" || base == "." || base == "/" {
		return "file"
	}
	return base
}
