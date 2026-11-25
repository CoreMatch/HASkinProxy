<div align="center">
  <h1>Armorial Skin Proxy</h1>
  <p><strong>让 CustomSkinLoader 客户端用上 Armorial 纹理。</strong></p>
  <p><a href="./README.md">English</a></p>
</div>

Armorial Skin Proxy 将 Armorial Yggdrasil API 转换为 Minecraft 客户端可用的 CustomSkinLoader 兼容响应。

它缓存档案与纹理数据，同时接入 Armorial 服务注册和 WebUI 交付流程。

## 为什么选择 Armorial Skin Proxy

- **无需改动上游即可转换。** 将 Yggdrasil 档案与纹理数据呈现为 CSL 格式。
- **缓存重复访问。** 在内存中保存档案与纹理，降低上游负载。
- **无需数据服务。** 不依赖数据库或 Redis。
- **兼顾旧客户端。** 透传旧版纹理删除请求格式。
- **自带客户端接入。** 提供配置 SDK 包与 Armorial 中继集成。

## 进一步了解

- [Armorial Wiki](https://wiki.mcnb.dev/) — 生态文档
- [Armorial API Contracts](https://github.com/CoreMatch/HA-Contract) — 生态 API 契约
- [CustomSkinLoader](https://github.com/xfl03/MCCustomSkinLoader) — 兼容的客户端项目

## 限制

客户端期望上游使用 `/yggdrasil-api` 基础路径。

## 许可证与来源

[AGPL-3.0](LICENSE)。来源：[CoreMatch/HASkinProxy](https://github.com/CoreMatch/HASkinProxy)。
