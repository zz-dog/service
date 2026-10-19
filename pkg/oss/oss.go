package oss

import (
	"context"
	"fmt"
	"io"
	"mime"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/google/uuid"
)

type Client struct {
	Client *oss.Client
	bucket string
	base   string
}

type ClientConfig struct {
	Region string

	Base      string
	Bucket    string
	KeyId     string
	KeySecret string
}

func NewClientClient(config ClientConfig) *Client {
	if config.Region == "" || config.Base == "" || config.Bucket == "" || config.KeyId == "" || config.KeySecret == "" {
		panic("oss config error")
	}
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.KeyId, config.KeySecret, "")).
		WithRegion(config.Region)
	client := oss.NewClient(cfg)
	return &Client{Client: client, bucket: config.Bucket, base: config.Base}
}

type PutInput struct {
	Ext    string
	Reader io.Reader
}

func (u *Client) Put(ctx context.Context, in PutInput, key string) (url string, err error) {

	_, err = u.Client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket:      oss.Ptr(u.bucket),            // 存储空间名称
		Key:         oss.Ptr(key),                 // 存储空间内的文件名称
		Body:        in.Reader,                    // 文件内容
		ContentType: oss.Ptr(ContentType(in.Ext)), // 文件类型
	})
	if err != nil {
		return "", err
	}
	return u.base + "/" + key, nil
}

// SignURL 生成对象的预签名读 URL(私有 Bucket 读取用),ttl 内有效
func (c *Client) SignURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	result, err := c.Client.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(key),
	}, oss.PresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// URL 对象的公网访问地址(Put 的返回值就是它;签名直传、引用历史 key 时单独用)
func (c *Client) URL(key string) string { return c.base + "/" + key }

// ContentType 扩展名 → MIME,查不到兜底 octet-stream
func ContentType(ext string) string {
	if ct := mime.TypeByExtension("." + ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
func GenKey(prefix, ext string) string {
	key := fmt.Sprintf("%s/%s.%s", time.Now().Format("20060102"), uuid.NewString(), ext)
	if prefix != "" {
		key = prefix + "/" + key
	}
	return key
}

func (c *Client) Presign(ctx context.Context, key, ext string, ttl time.Duration) (string, error) {
	result, err := c.Client.Presign(ctx, &oss.PutObjectRequest{
		Bucket:      oss.Ptr(c.bucket),         // 存储空间名称
		Key:         oss.Ptr(key),              // 存储空间内的文件名称
		ContentType: oss.Ptr(ContentType(ext)), // 文件类型
	},
		oss.PresignExpires(ttl),
	)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}
