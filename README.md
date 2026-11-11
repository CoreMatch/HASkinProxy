# HASkinProxy

A lightweight proxy service that converts **Yggdrasil API** logic into the **CustomSkinLoader (CSL)** compatible protocol format.

## Overview

`HASkinProxy` acts as a compatibility layer between Yggdrasil-compliant authentication services (such as [HRPAuth](https://github.com/CoreMatch/HRPAuth)) and Minecraft clients using CustomSkinLoader. 

While originally designed for the HA ecosystem (defined in [HA-Contract](https://github.com/CoreMatch/HA-Contract)), this proxy forwards standard Yggdrasil API calls. This means it is compatible with **any** authentication server that implements the Yggdrasil protocol, allowing seamless skin and cape loading without modifying the upstream service.

## Documentation

All interface documents and development specifications are unified in the root directory:
- [API Standard & Specification](https://github.com/CoreMatch/HA-Contract/blob/main/docs/api/README.md)
- [HASkinProxy Development Guide](https://github.com/CoreMatch/HA-Contract/blob/main/docs/dev/HASkinProxy/README.md)

## Key Features

- **CustomSkinAPI Compatibility**: Fully implements the standard CSL protocol.
- **Yggdrasil Integration**: Communicates with upstreams via standard Yggdrasil endpoints (`/api/profiles/minecraft` and `/sessionserver/session/minecraft/profile/{uuid}`).
- **Universal Support**: Works with HRPAuth or any other Yggdrasil-compatible server.
- **High Performance Caching**: Uses `freecache` for in-memory caching of profiles and texture data to reduce upstream load.
- **Auto-Config Generation**: Automatically generates a default `config.yaml` on the first run.
- **Lightweight**: Built with the Gin framework for high concurrency and low latency.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.24 or higher.

### Installation & Run

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd HASkinProxy
   ```

2. Build and run:
   ```bash
   go run main.go
   ```
   *On the first run, it will generate a `config.yaml` in the current directory.*

3. Configure your upstream:
   Edit `config.yaml` to point to your Yggdrasil-compatible service URL.

### Configuration

The `config.yaml` file includes the following sections:

```yaml
server:
  listen_addr: ":2702"        # Port for the proxy to listen on
  public_url: "http://localhost:2702" # Externally reachable base URL, relay source for HRPAuth
upstream:
  base_url: "http://localhost:2778" # HRPAuth root URL; Yggdrasil calls are sent to base_url + /yggdrasil-api automatically
  timeout: 10                # Upstream request timeout in seconds
  client_id: ""              # OAuth2 client id for the client_credentials grant
  client_secret: ""          # OAuth2 client secret for the client_credentials grant
cache:
  profile_ttl: 3600          # Profile cache duration (seconds)
  texture_ttl: 86400         # Texture cache duration (seconds)
  max_size_mb: 256           # Maximum cache size in MB
presence:
  enabled: true              # Register with HRPAuth via POST /services/presence
  name: "HASkinProxy"        # Service name in the presence registry
  ttl_seconds: 0             # Self-declared lifetime (seconds); <=0 means never expire
sdk:
  enabled: true              # Upload the embedded SDK package to HRPAuth via POST /services/sdk-packages
```

## Deployment facts

Authoritative deployment facts live in [`deploy-manifest.yaml`](deploy-manifest.yaml)
at the repo root (consumed by HA-Services-Deployer's orchestrator;
`manager: aapanel` means the process is panel-managed and the orchestrator
treats it as read-only):

- Listens on `:2702` (public); health probe `GET /health`.
- Depends on HRPAuth (upstream plus startup credential exchange); no database,
  no Redis, no data directories (in-memory cache only).
- Config file `config.yaml` — unversioned (no template version field).

## API Endpoints

- **GET `/{username}.json`**: Returns the player's CSL profile (skin/cape hashes).
- **GET `/textures/{hash}`**: Returns the raw texture image data.
- **POST `/{username}/texture/delete`**: Legacy-shaped delete passthrough — parses the legacy JSON body (`profile_id`, `texture_type`) and forwards the caller's `Authorization` header to the upstream `DELETE /texture/delete/{uuid}/{type}` route (see HA-Contract `docs/dev/HASkinProxy/README.md`).
- **GET `/health`**: Simple health check endpoint.

## WEBUI Integration

On startup, the proxy performs the presence (bonjour) handshake with HRPAuth (`POST /services/presence`, declaring the `webui-dash` frontend area) and registers a relay rule (`POST /services/relay`) so the CSL API is served through the main service origin under the `/csl/` prefix. Requires `server.public_url` to be set to an address reachable by HRPAuth. The CustomSkinLoader setup page for the dashboard is shipped as a compile-time SDK package (the embedded `sdk/` tree), uploaded to HRPAuth via `POST /services/sdk-packages` and aggregated into the frontend build by the WEBUI SDK handler — no runtime page route is served for it.

## Architecture

1. **Request**: Client requests `Player.json`.
2. **Lookup**: Proxy fetches UUID via `POST /api/profiles/minecraft`.
3. **Profile**: Proxy fetches Yggdrasil profile via `GET /sessionserver/session/minecraft/profile/{uuid}`.
4. **Transform**: Proxy extracts texture hashes and formats them into CSL JSON.
5. **Cache**: Result is cached to speed up subsequent requests.

## Related Projects

- **HRPAuth**: [https://github.com/CoreMatch/HRPAuth](https://github.com/CoreMatch/HRPAuth)
- **HA-Contract**: [https://github.com/CoreMatch/HA-Contract](https://github.com/CoreMatch/HA-Contract)
- **CustomSkinLoader**: [https://github.com/xfl03/MCCustomSkinLoader](https://github.com/xfl03/MCCustomSkinLoader)

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).
