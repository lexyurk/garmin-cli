# garmin

Fast, ergonomic Garmin Connect CLI written in Go.

> Inspired by [garmin-connect-cli](https://github.com/eddmann/garmin-connect-cli) (eddmann) but rewritten for speed and better UX.

## Features

- **Fast** — native Go binary, instant startup
- **All your Garmin data** — activities, sleep, HR, stress, body battery, training metrics, fitness age
- **Plan and manage** — build structured workouts, schedule them, manage gear/shoes, edit activities
- **LLM-friendly by default** — Markdown output that’s easy to paste into chats/prompts
- **Multiple output formats** — markdown (default), tables, human, JSON
- **Script-friendly when needed** — JSON output, composable with `jq`, pipes, standard Unix tools

## Installation

### Download a release

Grab a prebuilt binary for your OS/arch from the
[Releases](https://github.com/lexyurk/garmin-cli/releases) page (linux, macOS,
and Windows; amd64 and arm64), then extract and put `garmin` on your `PATH`:

```bash
tar -xzf garmin-cli_*_linux_amd64.tar.gz
sudo mv garmin /usr/local/bin/
garmin version
```

### Go install

```bash
# latest release (recommended — reports a real version)
go install github.com/lexyurk/garmin-cli/cmd/garmin@latest

# a specific release
go install github.com/lexyurk/garmin-cli/cmd/garmin@v0.1.0
```

### From source

```bash
git clone https://github.com/lexyurk/garmin-cli
cd garmin-cli
make install

# Ensure $GOPATH/bin is on your PATH
export PATH="$(go env GOPATH)/bin:$PATH"
```

## Quick Start

```bash
# Authenticate with Garmin Connect
garmin auth login

# One-shot snapshot: health + training + latest activity
garmin today

# Today's health data
garmin health summary
garmin health sleep
garmin health body-battery
garmin health heart-rate
garmin health stress
garmin health steps
garmin health spo2
garmin health respiration
garmin health intensity-minutes

# Recent activities
garmin activities list --limit 10
garmin activities get <activity-id>
garmin activities splits <activity-id>
garmin activities export <activity-id> --type gpx --out activity.gpx
garmin activities update <activity-id> --name "Tempo run" --type running
garmin activities delete <activity-id>

# Training metrics
garmin training status
garmin training readiness
garmin training vo2max
garmin training hrv
garmin training fitness-age

# Profile
garmin profile

# Gear (shoes, bikes)
garmin gear list                 # active gear
garmin gear list --all --stats   # include retired + cumulative mileage
garmin gear add --name "Pegasus 40" --make Nike --max-km 800
garmin gear link "Pegasus 40" --last           # tag your most recent run
garmin gear link "Pegasus 40" <activity-id>    # or a specific activity
garmin gear for-activity <activity-id>
garmin gear retire "Pegasus 40"

# Workouts (plan and build)
garmin workouts list
garmin workouts create --name "4x800m" \
  --step "warmup 10min" \
  --step "4x(interval 800m; recovery 2min)" \
  --step "cooldown 5min"
garmin workouts schedule <workout-id> --date 2026-06-01
garmin workouts delete <workout-id>

# Training calendar
garmin calendar
garmin calendar --month 2026-06 --type workout

# Weight & body composition
garmin weight log 74.5
garmin weight list --days 30
garmin weight latest

# Records, predictions, devices
garmin records
garmin training race-predictions
garmin devices list

# Check auth status
garmin auth status

# Switch output to JSON (for scripts)
garmin health sleep --format json

# Time ranges
garmin health sleep --days 7
garmin activities list --days 7 --limit 50
garmin activities list --from 2026-01-01 --to 2026-02-01 --limit 50
garmin activities list --date 2026-02-16 --limit 50
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
| `--config-dir` | `-c` | Config directory (tokens, settings) (deprecated alias: `--config`) |
| `--profile` | `-p`  | Named profile to use                     |

### Commands

| Command                  | Description                            |
|--------------------------|----------------------------------------|
| `today`                  | One-shot snapshot (health + training + activity) |
| `auth login`             | Login to Garmin Connect                |
| `auth status`            | Check authentication status            |
| `auth logout`            | Clear stored tokens                    |
| `health sleep`           | Sleep data                             |
| `health summary`         | Daily health summary                   |
| `health heart-rate`      | Heart rate data                        |
| `health steps`           | Step count                             |
| `health stress`          | Stress levels                          |
| `health body-battery`    | Body battery                           |
| `health spo2`            | Pulse ox (SpO2)                        |
| `health respiration`     | Respiration rate                       |
| `health intensity-minutes` | Moderate/vigorous intensity minutes  |
| `activities list`        | List activities                        |
| `activities get <id>`    | Get activity details                   |
| `activities splits <id>` | Get activity splits/laps               |
| `activities export <id>` | Download activity GPX/TCX/original     |
| `activities update <id>` | Rename / set type / set description     |
| `activities delete <id>` | Delete an activity                     |
| `activities weather <id>`| Weather recorded during an activity    |
| `training status`        | Training status                        |
| `training readiness`     | Training readiness score               |
| `training vo2max`        | VO2 max estimates                      |
| `training hrv`           | Heart rate variability                 |
| `training fitness-age`   | Fitness age (health age) estimate      |
| `training race-predictions` | Predicted 5K/10K/half/marathon times |
| `profile`                | Show your Garmin Connect profile       |
| `records`                | Personal records (PRs)                 |
| `gear list`              | List gear (shoes, bikes)               |
| `gear get <gear>`        | Gear details with cumulative stats     |
| `gear stats <gear>`      | Cumulative distance/activities         |
| `gear activities <gear>` | Activities recorded with a gear item   |
| `gear add`               | Add a gear item                        |
| `gear retire <gear>`     | Retire / restore a gear item           |
| `gear link <gear> <id>`  | Assign gear to an activity (or `--last`) |
| `gear unlink <gear> <id>`| Remove gear from an activity           |
| `gear for-activity <id>` | Show gear linked to an activity        |
| `gear set-default <uuid>`| Set/clear default gear per activity type |
| `workouts list`          | List saved workouts                    |
| `workouts get <id>`      | Get workout details                    |
| `workouts create`        | Build and create a structured workout  |
| `workouts update <id>`   | Update workout name/description        |
| `workouts delete <id>`   | Delete a workout                       |
| `workouts schedule <id>` | Schedule a workout onto a date         |
| `workouts unschedule <schedule-id>` | Remove a scheduled occurrence |
| `workouts export <id>`   | Download a workout's FIT file          |
| `calendar`               | View the training calendar             |
| `weight log <kg>`        | Log a weigh-in                         |
| `weight list`            | List weigh-ins over a range            |
| `weight latest`          | Most recent weigh-in                   |
| `devices list`           | List registered Garmin devices         |
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

## Releasing

Releases are automated with [GoReleaser](https://goreleaser.com). Pushing a
semver tag triggers the `Release` workflow, which cross-compiles binaries and
publishes a GitHub Release with archives, `checksums.txt`, and a changelog.

```bash
# Validate the config and dry-run a build locally first
make release-check
make snapshot          # builds into ./dist (nothing is published)

# Cut a release
git tag v0.1.0
git push origin v0.1.0
```

The version is injected into `garmin version` via ldflags, so released and
`go install`ed binaries report their real version (local `make build` uses
`git describe`).

## License

MIT
