// Package nacos 实现基于 Nacos 配置中心的远程配置拉取与变更监听，
// 与 discovery/nacos（服务注册发现）相对，是配置管理的"另一半"。
package nacos

import (
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
)

// ConfigClient Nacos 配置中心客户端
type ConfigClient struct {
	client config_client.IConfigClient
}

// NewConfigClient 创建配置中心客户端；Nacos 未启用时返回 (nil, nil)，此时 Pull/Load/Watch 均为空操作
func NewConfigClient() (*ConfigClient, error) {
	if !global.Conf.Nacos.Enabled {
		return nil, nil
	}

	// 创建 Nacos 配置客户端
	client, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig: &constant.ClientConfig{
			NamespaceId: global.Conf.Nacos.Namespace,
			Username:    global.Conf.Nacos.Username,
			Password:    global.Conf.Nacos.Password,
			LogLevel:    "error",
		},

		// 服务注册配置
		ServerConfigs: []constant.ServerConfig{*constant.NewServerConfig(
			global.Conf.Nacos.ServerAddr,
			global.Conf.Nacos.ServerPort,
		)},
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Nacos 配置客户端失败: %w", err)
	}
	return &ConfigClient{client: client}, nil
}

// configParam 按配置组装请求参数：Data ID 默认 service-config.yaml，分组与服务注册保持一致
func configParam() vo.ConfigParam {
	dataId := global.Conf.Nacos.ConfigDataId
	if dataId == "" {
		dataId = "service-config.yaml"
	}
	group := global.Conf.Nacos.GroupName
	if group == "" {
		group = "DEFAULT_GROUP"
	}
	return vo.ConfigParam{DataId: dataId, Group: group}
}

// Pull 拉取远程配置内容（YAML 文本）；Nacos 未启用时返回空串
func (c *ConfigClient) Pull() (string, error) {
	if c == nil {
		return "", nil
	}
	param := configParam()
	content, err := c.client.GetConfig(param)
	if err != nil {
		return "", fmt.Errorf("拉取 Nacos 配置失败 (DataId=%s): %w", param.DataId, err)
	}
	return content, nil
}

// Load 拉取远程配置并合并到全局配置（远端键覆盖本地同名键，环境变量仍最高优先）。
// Nacos 未启用或远程配置为空时静默使用本地配置；拉取/合并失败返回错误，由调用方决定是否 fail-fast。
// 需在 global.InitZap 之后调用（内部使用全局日志）。
func (c *ConfigClient) Load() error {
	content, err := c.Pull()
	if err != nil {
		return err
	}
	if content == "" {
		if c != nil {
			global.Logger.Warn("Nacos 远程配置为空（Data ID 不存在或未发布内容），使用本地配置启动")
		}
		return nil
	}
	if err := global.MergeRemoteConfig(content); err != nil {
		return err
	}
	global.Logger.Info("已加载 Nacos 远程配置", zap.String("data_id", configParam().DataId))
	return nil
}

// Watch 监听远程配置变更，内容变化时回调 onChange（content 为最新内容，空串表示配置已删除）
func (c *ConfigClient) Watch(onChange func(content string)) error {
	if c == nil {
		return nil
	}
	param := configParam()
	param.OnChange = func(_, _, _, data string) {
		onChange(data)
	}
	return c.client.ListenConfig(param)
}
