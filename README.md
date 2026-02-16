# garmin

Fast, ergonomic Garmin Connect CLI written in Go.

> Inspired by [garmin-connect-cli](https://github.com/eddmann/garmin-connect-cli) (eddmann) but rewritten for speed and better UX.

## Features

- **Fast** — native Go binary, instant startup
- **All your Garmin data** — activities, sleep, HR, stress, body battery, training metrics
- **LLM-friendly by default** — Markdown output that’s easy to paste into chats/prompts
- **Multiple output formats** — markdown (default), tables, human, JSON
- **Script-friendly when needed** — JSON output, composable with `jq`, pipes, standard Unix tools

## Installation

### From source

```bash
git clone https://github.com/lexyurk/garmin-cli
cd garmin-cli
make install

# Ensure $GOPATH/bin is on your PATH
export PATH="$(go env GOPATH)/bin:$PATH"
```

### Go install

```bash
go install github.com/lexyurk/garmin-cli/cmd/garmin@latest
```

## Quick Start

```bash
# Authenticate with Garmin Connect
garmin auth login

# Today's health data
garmin health summary
garmin health sleep
garmin health body-battery
garmin health heart-rate
garmin health stress
garmin health steps

# Recent activities
garmin activities list --limit 10
garmin activities get <activity-id>
garmin activities splits <activity-id>

# Training metrics
garmin training status
garmin training readiness
garmin training vo2max
garmin training hrv

# Check auth status
garmin auth status

# Switch output to JSON (for scripts)
garmin health sleep --format json

# Time ranges
garmin health sleep --days 7
garmin activities list --after 2026-01-01 --before 2026-02-01 --limit 50
```

## Shell completion

```bash
# bash
source <(garmin completion bash)

# zsh/fish/powershell are also supported:
garmin completion zsh
garmin completion fish
garmin completion powershell
```

### Non-interactive login (recommended for CI)

```bash
printf '%s' "$GARMIN_PASSWORD" | garmin auth login --email "$GARMIN_EMAIL" --password-stdin
```

## Command Reference

### Global Options

| Flag        | Short | Description                              |
|-------------|-------|------------------------------------------|
| `--format`  | `-f`  | Output format: markdown, table, human, json |
| `--verbose` | `-v`  | Verbose output (HTTP request logs to stderr) |
| `--quiet`   | `-q`  | Suppress non-essential output            |
| `--config-dir` | `-c` | Config directory (tokens, settings)   |
| `--profile` | `-p`  | Named profile to use                     |

### Commands

| Command                  | Description                            |
|--------------------------|----------------------------------------|
| `auth login`             | Login to Garmin Connect                |
| `auth status`            | Check authentication status            |
| `auth logout`            | Clear stored tokens                    |
| `health sleep`           | Sleep data                             |
| `health summary`         | Daily health summary                   |
| `health heart-rate`      | Heart rate data                        |
| `health steps`           | Step count                             |
| `health stress`          | Stress levels                          |
| `health body-battery`    | Body battery                           |
| `activities list`        | List activities                        |
| `activities get <id>`    | Get activity details                   |
| `activities splits <id>` | Get activity splits/laps               |
| `training status`        | Training status                        |
| `training readiness`     | Training readiness score               |
| `training vo2max`        | VO2 max estimates                      |
| `training hrv`           | Heart rate variability                 |
| `completion <shell>`     | Generate shell completion scripts      |
| `version`                | Print version                          |

## Configuration

Config stored in `~/.config/garmin/config.toml`.

Supported keys:

```toml
# ~/.config/garmin/config.toml
format = "markdown" # markdown|table|human|json
profile = "default"
```

Tokens stored in:

```
~/.config/garmin/tokens/<profile>/oauth1_token.json
~/.config/garmin/tokens/<profile>/oauth2_token.json
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `GARMIN_CONFIG_DIR` | Overrides config directory (same as `--config-dir`) |
| `GARMIN_PROFILE` | Default profile (overridden by `--profile`) |
| `GARMIN_FORMAT` | Default output format (overridden by `--format`) |
| `GARMIN_EMAIL` | Default email for `garmin auth login` (overridden by `--email`) |
| `GARMIN_PASSWORD` | Default password for `garmin auth login` (overridden by `--password` / `--password-stdin`) |

## Authentication notes (SSO, MFA)

- `garmin auth login` uses Garmin SSO to obtain OAuth tokens for `connectapi.garmin.com`.
- If you omit `--email` and/or `--password`, you’ll be prompted (when running in a TTY).
- For CI / non-interactive usage, prefer `--password-stdin` so the password isn’t visible in process args:
  - `printf '%s' "$GARMIN_PASSWORD" | garmin auth login --email "$GARMIN_EMAIL" --password-stdin`
- If your account requires MFA, you’ll be prompted for the code.
  - For non-interactive usage, pass `--mfa-code <code>`.
- Tokens are refreshed automatically when needed (OAuth2 is exchanged again via OAuth1, like the `garth` Python library).

## Troubleshooting

### `not authenticated`

Run:

```bash
garmin auth login
```

### Stuck / expired tokens

Try:

```bash
garmin auth logout
garmin auth login
```

## Development

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
make install  # Install to $GOPATH/bin
```

## License

MIT
