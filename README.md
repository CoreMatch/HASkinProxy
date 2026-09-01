# HASkinProxy

A lightweight proxy service that converts **Yggdrasil API** logic into the **CustomSkinLoader (CSL)** compatible protocol format.

## Overview

`HASkinProxy` acts as a compatibility layer between Yggdrasil-compliant authentication services (such as [HRPAuth](https://github.com/CoreMatch/HRPAuth)) and Minecraft clients using CustomSkinLoader. 

While originally designed for the HA ecosystem (defined in [HA-Contract](https://github.com/CoreMatch/HA-Contract)), this proxy forwards standard Yggdrasil API calls. This means it is compatible with **any** authentication server that implements the Yggdrasil protocol, allowing seamless skin and cape loading without modifying the upstream service.

## Key Features

- **CustomSkinAPI Compatibility**: Fully implements the standard CSL protocol.
- **Yggdrasil Integration**: Communicates with upstreams via standard Yggdrasil endpoints (`/api/profiles/minecraft` and `/sessionserver/session/minecraft/profile/{uuid}`).
- **Universal Support**: Works with HRPAuth or any other Yggdrasil-compatible server.
- **High Performance Caching**: Uses `freecache` for in-memory caching of profiles and texture data to reduce upstream load.
- **Auto-Config Generation**: Automatically generates a default `config.yaml` on the first run.
- **Lightweight**: Built with the Gin framework for high concurrency and low latency.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.20 or higher.

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
upstream:
  base_url: "http://localhost:2778" # Your Yggdrasil service URL (e.g., HRPAuth)
  timeout: 10                # Upstream request timeout in seconds
cache:
  profile_ttl: 3600          # Profile cache duration (seconds)
  texture_ttl: 86400         # Texture cache duration (seconds)
  max_size_mb: 256           # Maximum cache size in MB
presence:
  enabled: true              # Register with HRPAuth via POST /services/presence
  name: "HASkinProxy"        # Service name in the presence registry
  ttl_seconds: 0             # Self-declared lifetime (seconds); <=0 means never expire
```

## API Endpoints

- **GET `/{username}.json`**: Returns the player's CSL profile (skin/cape hashes).
- **GET `/textures/{hash}`**: Returns the raw texture image data.
- **GET `/health`**: Simple health check endpoint.

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
