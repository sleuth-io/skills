# Skills

A CLI tool for managing Sleuth skills - reusable units of AI agent behavior.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/sleuth-io/skills/main/install.sh | bash
```

This downloads and installs the pre-built binary for your platform.

## Getting Started

Initialize your skills configuration:

```bash
skills init
```

This creates a configuration file in your home directory.

## Usage

### Adding Skills

Add a skill from a local directory or zip file:

```bash
skills add /path/to/skill
skills add skill.zip
```

### Managing Dependencies

Generate a lock file from your requirements:

```bash
skills lock
```

Install skills from the lock file:

```bash
skills install
```

### Help

View all available commands:

```bash
skills --help
skills <command> --help
```

## Documentation

- [Repository Spec](docs/repository-spec.md) - Skills repository structure
- [Metadata Spec](docs/metadata-spec.md) - Skill metadata format
- [Requirements Spec](docs/requirements-spec.md) - Dependency requirements
- [Lock Spec](docs/lock-spec.md) - Lock file format

## License

See LICENSE file for details.

---

## Development

<details>
<summary>Click to expand development instructions</summary>

### Prerequisites

Go 1.25 or later is required. Install using [gvm](https://github.com/moovweb/gvm):

```bash
# Install gvm
bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)

# Install Go (use go1.4 as bootstrap if needed)
gvm install go1.4 -B
gvm use go1.4
export GOROOT_BOOTSTRAP=$GOROOT
gvm install go1.25
gvm use go1.25 --default
```

### Building from Source

```bash
make init           # First time setup (install tools, download deps)
make build          # Build binary
make install        # Install to GOPATH/bin
```

### Testing

```bash
make test           # Run tests with race detection
make format         # Format code with gofmt
make lint           # Run golangci-lint
make prepush        # Run before pushing (format, lint, test, build)
```

### Releases

Tag and push to trigger automated release via GoReleaser:

```bash
git tag v0.1.0
git push origin v0.1.0
```

</details>
