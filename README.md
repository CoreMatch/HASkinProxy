<div align="center">
  <h1>HASkinProxy</h1>
  <p><strong>Bring HA textures to CustomSkinLoader clients.</strong></p>
  <p><a href="./README.zh-CN.md">简体中文</a></p>
</div>

HASkinProxy translates the HA Yggdrasil API into CustomSkinLoader-compatible responses for Minecraft clients.

It caches profiles and texture data while integrating with HRPAuth service registration and WebUI delivery.

## Why HASkinProxy

- **Translate without changing upstream.** Present Yggdrasil profile and texture data in CSL format.
- **Cache for repeat visits.** Keep profiles and textures in memory to reduce upstream load.
- **Run without data services.** Require no database or Redis.
- **Keep legacy clients working.** Pass through the legacy texture-delete request shape.
- **Include client setup.** Ship a setup SDK package and HRPAuth relay integration.

## Learn more

- [HA Wiki](https://wiki.mcnb.dev/) — ecosystem documentation
- [HA-Contract](https://github.com/CoreMatch/HA-Contract) — ecosystem API contracts
- [CustomSkinLoader](https://github.com/xfl03/MCCustomSkinLoader) — compatible client project

## Limit

The client expects the upstream at its `/yggdrasil-api` base path.

## License & provenance

[AGPL-3.0](LICENSE). Source: [CoreMatch/HASkinProxy](https://github.com/CoreMatch/HASkinProxy).
