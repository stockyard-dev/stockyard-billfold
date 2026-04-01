# Stockyard Billfold

**Invoicing for freelancers — clients, PDF invoices, payment status, simple reports**

Part of the [Stockyard](https://stockyard.dev) family of self-hosted developer tools.

## Quick Start

```bash
docker run -p 9270:9270 -v billfold_data:/data ghcr.io/stockyard-dev/stockyard-billfold
```

Or with docker-compose:

```bash
docker-compose up -d
```

Open `http://localhost:9270` in your browser.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9270` | HTTP port |
| `DATA_DIR` | `./data` | SQLite database directory |
| `BILLFOLD_LICENSE_KEY` | *(empty)* | Pro license key |

## Free vs Pro

| | Free | Pro |
|-|------|-----|
| Limits | 3 clients, 10 invoices | Unlimited clients and invoices |
| Price | Free | $2.99/mo |

Get a Pro license at [stockyard.dev/tools/](https://stockyard.dev/tools/).

## Category

Creator & Small Business

## License

Apache 2.0
