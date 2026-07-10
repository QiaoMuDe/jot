# 密钥 Base64 编码存储 Spec

## Why

当前项目中的 API Key、搜索 Key 等敏感密钥以明文形式直接存储在 SQLite 数据库中，打开 DB 文件即可一览无余。通过 Base64 编码存储，防止日常随意浏览 DB 文件时一眼看出密码。

## What Changes

- 新增 `internal/services/crypto.go`，提供基于前缀标识的 `EncodeB64()` / `DecodeB64()` 工具函数
- 编码后格式：`(zk)<standard_base64>`，通过前缀 `(zk)` 标记已编码状态
- 修改 `AIService.SaveConfig()`：写入 DB 前对 3 个密钥字段做 Base64 编码
- 修改 `AIService.GetConfig()`：从 DB 读取后对 3 个密钥字段做 Base64 解码
- 修改 `ProfileService` 的 Create/Update/List 方法：对 `api_key` 做编解码
- 修改 `app.go` 中测试连接和获取模型等函数：从 DB 读取后先解码再传入下游
- 应用启动时迁移已有的明文密钥为带 `(zk)` 前缀的 Base64 编码

## Impact

- 前端完全无感知，所有编解码在后端完成
- 现有的 settings 表和 api_profiles 表中的存量数据需要迁移

## ADDED Requirements

### Requirement: Base64 编解码工具函数

The system SHALL provide `EncodeB64(s string) string` 和 `DecodeB64(s string) string` 两个函数。

编码后格式为 `(zk)<base64_encoded>`，其中 `(zk)` 是固定前缀，用于标识已编码状态。

#### Scenario: 标准编码解码
- **WHEN** 调用 `EncodeB64("sk-test-key")`
- **THEN** 返回 `"(zk)c2stdZXN0LWtleQ=="`
- **WHEN** 调用 `DecodeB64("(zk)c2stdZXN0LWtleQ==")`
- **THEN** 返回 `"sk-test-key"`

#### Scenario: 明文兼容解码
- **WHEN** 调用 `DecodeB64("sk-test-key")`（无 `(zk)` 前缀，明文）
- **THEN** 返回原字符串 `"sk-test-key"`（兼容迁移前的存量数据）

#### Scenario: 空字符串处理
- **WHEN** 调用 `EncodeB64("")` 或 `DecodeB64("")`
- **THEN** 返回空字符串

### Requirement: AI 配置存储编解码

The system SHALL 在 `AIService.SaveConfig()` 中对 `APIKey`、`TavilyAPIKey`、`ZhihuAccessSecret` 三个字段编码后再存入 DB。

#### Scenario: 保存 AI 配置
- **WHEN** 用户保存 AI 配置
- **THEN** 密钥字段以 `(zk)<base64>` 格式写入 settings 表

#### Scenario: 读取 AI 配置
- **WHEN** 前端请求 AI 配置
- **THEN** 密钥字段以解码后的明文返回给前端

### Requirement: API 预设编解码

The system SHALL 在 `ProfileService` 的 Create/Update/Switch/List 方法中对 `APIKey` 做编解码。

#### Scenario: 创建/更新预设
- **WHEN** 用户创建或更新 API 预设
- **THEN** `api_key` 以 `(zk)<base64>` 格式写入 api_profiles 表

#### Scenario: 读取预设列表
- **WHEN** 前端获取预设列表
- **THEN** 每个预设的 `api_key` 以解码后的明文返回

### Requirement: 测试连接和获取模型

The system SHALL 在 `app.go` 的测试连接和获取模型方法中：
- 如果密钥直接由前端传入（明文），直接使用
- 如果密钥从 DB 读取（经服务层已解码），也是明文直接使用

#### Scenario: 测试连接
- **WHEN** 用户点击「测试连接」按钮，传入明文密钥
- **THEN** 直接使用传入的明文密钥测试

#### Scenario: 获取模型列表
- **WHEN** 用户获取模型列表，传入明文密钥
- **THEN** 直接使用传入的明文密钥

### Requirement: 存量数据迁移

The system SHALL 在应用启动时自动检测并迁移已有的明文密钥。

#### Scenario: 启动迁移
- **WHEN** 应用启动，扫描 settings 表和 api_profiles 表
- **THEN** 对每个非空值的密钥字段，检查是否以 `(zk)` 开头
- **THEN** 无 `(zk)` 前缀 → 视为明文 → 执行 `EncodeB64()` 后写回
- **THEN** 已有 `(zk)` 前缀 → 跳过，确保幂等

## MODIFIED Requirements

### Requirement: AI 服务配置读写

**修改前**：`GetConfig()` 和 `SaveConfig()` 直接读写明文
**修改后**：`SaveConfig()` 编码后写入，`GetConfig()` 解码后返回

### Requirement: API 配置预设 CRUD

**修改前**：`api_key` 明文存储在 api_profiles 表
**修改后**：`api_key` 以 `(zk)<base64>` 格式存储，读取时解码

## REMOVED Requirements

无
