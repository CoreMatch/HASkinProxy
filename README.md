<div align="center">
  <h1>Armorial Skin Proxy</h1>
  <p><strong>Bring Armorial textures to CustomSkinLoader clients.</strong></p>
  <p><a href="./README.zh-CN.md">简体中文</a></p>
</div>

Armorial Skin Proxy translates the Armorial Yggdrasil API into CustomSkinLoader-compatible responses for Minecraft clients.

It caches profiles and texture data while integrating with Armorial service registration and WebUI delivery.

## Why Armorial Skin Proxy

- **Translate without changing upstream.** Present Yggdrasil profile and texture data in CSL format.
- **Cache for repeat visits.** Keep profiles and textures in memory to reduce upstream load.
- **Run without data services.** Require no database or Redis.
- **Keep legacy clients working.** Pass through the legacy texture-delete request shape.
- **Include client setup.** Ship a setup SDK package and Armorial relay integration.

## Learn more

- [Armorial Wiki](https://wiki.mcnb.dev/) — ecosystem documentation
- [Armorial API Contracts](https://github.com/CoreMatch/HA-Contract) — ecosystem API contracts
- [CustomSkinLoader](https://github.com/xfl03/MCCustomSkinLoader) — compatible client project

## Limit

The client expects the upstream at its `/yggdrasil-api` base path.

## License & provenance

[AGPL-3.0](LICENSE). Source: [CoreMatch/HASkinProxy](https://github.com/CoreMatch/HASkinProxy).
