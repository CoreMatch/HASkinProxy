# HASkinProxy

一个轻量级的代理服务，将 **Yggdrasil API** 逻辑转换为 **CustomSkinLoader (CSL)** 兼容的协议格式。

## 项目简介

`HASkinProxy` 作为兼容 Yggdrasil 标准的验证服务（如 [HRPAuth](https://github.com/CoreMatch/HRPAuth)）与使用 CustomSkinLoader 的 Minecraft 客户端之间的兼容层。

虽然本项目最初是为 HA 生态系统（定义于 [HA-Contract](https://github.com/CoreMatch/HA-Contract)）设计的，但由于它转发的是标准的 Yggdrasil API 调用，这意味着它**兼容任何实现了 Yggdrasil 协议的验证服务器**。它允许在不修改上游服务的情况下，实现无缝的皮肤和披风加载。

## 核心特性

- **兼容 CustomSkinAPI**: 完整实现标准的 CSL 协议。
- **Yggdrasil 集成**: 通过标准的 Yggdrasil 接口（`/api/profiles/minecraft` 和 `/sessionserver/session/minecraft/profile/{uuid}`）与上游通信。
- **通用性**: 支持 HRPAuth 或任何其他兼容 Yggdrasil 的服务器。
- **高性能缓存**: 使用 `freecache` 进行内存级缓存（玩家信息和材质数据），大幅降低上游负载。
- **配置自动生成**: 首次运行会自动在当前目录生成默认的 `config.yaml`。
- **轻量高效**: 基于 Gin 框架构建，支持高并发及低延迟。

## 快速入门

### 前置要求

- [Go](https://golang.org/dl/) 1.20 或更高版本。

### 安装与运行

1. 克隆仓库：
   ```bash
   git clone <repository-url>
   cd HASkinProxy
   ```

2. 运行项目：
   ```bash
   go run main.go
   ```
   *首次运行时，程序会在当前目录下生成 `config.yaml`。*

3. 配置上游：
   修改 `config.yaml` 中的 `base_url` 指向您的 Yggdrasil 兼容服务地址。

### 配置说明

`config.yaml` 文件包含以下配置项：

```yaml
server:
  listen_addr: ":2702"        # 代理服务监听端口
upstream:
  base_url: "http://localhost:2778" # Yggdrasil 服务地址 (例如 HRPAuth)
  timeout: 10                # 上游请求超时时间（秒）
cache:
  profile_ttl: 3600          # 玩家信息缓存时长（秒）
  texture_ttl: 86400         # 材质数据缓存时长（秒）
  max_size_mb: 256           # 最大缓存占用空间 (MB)
```

## API 接口

- **GET `/{username}.json`**: 获取玩家的 CSL 信息（包含皮肤/披风哈希）。
- **GET `/textures/{hash}`**: 获取原始材质图像数据。
- **GET `/health`**: 基础健康检查接口。

## 核心流程

1. **请求**: 客户端请求 `Player.json`。
2. **查询**: 代理通过 `POST /api/profiles/minecraft` 获取 UUID。
3. **获取**: 代理通过 `GET /sessionserver/session/minecraft/profile/{uuid}` 获取 Yggdrasil 资料。
4. **转换**: 代理提取材质哈希并将其格式化为 CSL JSON。
5. **缓存**: 结果被缓存以加速后续请求。

## 相关项目

- **HRPAuth**: [https://github.com/CoreMatch/HRPAuth](https://github.com/CoreMatch/HRPAuth)
- **HA-Contract**: [https://github.com/CoreMatch/HA-Contract](https://github.com/CoreMatch/HA-Contract)
- **CustomSkinLoader**: [https://github.com/xfl03/MCCustomSkinLoader](https://github.com/xfl03/MCCustomSkinLoader)

## 开源协议

本项目采用 [GNU Affero General Public License v3.0](LICENSE) 协议。
