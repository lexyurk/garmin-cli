# garmin-cli

Fast, ergonomic Garmin Connect CLI written in Go.

> Inspired by [garmin-connect-cli](https://github.com/eddmann/garmin-connect-cli) (eddmann) but rewritten for speed and better UX.

## Features

- **Fast** — native Go binary, instant startup
- **All your Garmin data** — activities, sleep, HR, stress, body battery, training metrics
- **Script-friendly** — JSON output, composable with `jq`, pipes, standard Unix tools
- **Multiple output formats** — JSON for scripts, tables for terminals, human-readable for quick glances

## Installation

### From source

```bash
git clone https://github.com/lexyurk/garmin-cli
cd garmin-cli
make install
```

### Go install

```bash
go install github.com/lexyurk/garmin-cli/cmd/garmin@latest
```

## Quick Start

```bash
# Authenticate with Garmin Connect
garmin-cli auth login

# Today's health data
garmin-cli health sleep
garmin-cli health body-battery
garmin-cli health heart-rate
garmin-cli health stress
garmin-cli health steps

# Recent activities
garmin-cli activities list --limit 10
garmin-cli activities get <activity-id>
garmin-cli activities splits <activity-id>

# Training metrics
garmin-cli training status
garmin-cli training readiness
garmin-cli training vo2max
garmin-cli training hrv

# Check auth status
garmin-cli auth status
```

## Command Reference

### Global Options

| Flag        | Short | Description                              |
|-------------|-------|------------------------------------------|
| `--format`  | `-f`  | Output format: json, table, human        |
| `--verbose` | `-v`  | Verbose output                           |
| `--quiet`   | `-q`  | Suppress non-essential output            |
| `--config`  | `-c`  | Path to config file                      |
| `--profile` | `-p`  | Named profile to use                     |

### Commands

| Command                  | Description                            |
|--------------------------|----------------------------------------|
| `auth login`             | Login to Garmin Connect                |
| `auth status`            | Check authentication status            |
| `auth logout`            | Clear stored tokens                    |
| `health sleep`           | Sleep data                             |
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

## Configuration

Config stored in `~/.config/garmin-cli/config.toml` (planned).

Tokens stored in `~/.config/garmin-cli/tokens/`.

### Environment Variables

| Variable          | Description             |
|-------------------|-------------------------|
| `GARMIN_EMAIL`    | Garmin Connect email    |
| `GARMIN_PASSWORD` | Garmin Connect password |
| `GARMIN_FORMAT`   | Default output format   |
| `GARMIN_PROFILE`  | Default profile name    |

## Development

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
make install  # Install to $GOPATH/bin
```

## License

MIT
