<div align="center"><img src="logo.svg" alt="" width="64" height="64"></div>
<h1 align="center">Swopcart Server</h1>
<div align="center">
  <img alt="Codecov" src="https://img.shields.io/codecov/c/gh/swopcart/server">
</div>

Swopcart is a self-hosted personal library, like [Jellyfin], but for your games.

[Jellyfin]: https://jellyfin.org/

## Development

Install [Docker], [Go] and [Node], then install dependencies:

```bash
go get ./...
npm -C frontend install
```

Common development tasks using [Just]:

```bash
# Start local PostgreSQL database
just local-db              # Run in foreground (see logs)
just start-local-db        # Run in background (-d)
just stop-local-db         # Stop the database

# Run development servers (with hot reload)
just serve                 # Both backend and frontend in parallel
just serve-backend         # Backend only (uses air for hot reload)
just serve-frontend        # Frontend only (Vite dev server)

# Testing and quality
just test                  # Run all Go tests
just lint                  # Lint Go and TypeScript
just fmt                   # Format Go and TypeScript
```

[Docker]: https://www.docker.com/
[Go]: https://go.dev/
[Node]: https://nodejs.org/en
[Just]: https://just.systems/

## Contribute

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for guidelines on submitting issues and pull requests.

## Licence

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). See [LICENSE](LICENSE) for the full license text.

In short: you're free to use, modify, and distribute this software, but if you run a modified version on a server, you must make the source code available to users.
