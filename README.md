# skillhub

A lightweight CLI package manager for AI agent skills.

Search, install, run, update, and remove skills from registries with a single command.

## Installation

### Binary (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/jayl2kor/skillhub/master/install.sh | sh
```

Or with `wget`:

```bash
wget -qO- https://raw.githubusercontent.com/jayl2kor/skillhub/master/install.sh | sh
```

To install to a custom directory:

```bash
INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/jayl2kor/skillhub/master/install.sh | sh
```

### From source

```bash
go install github.com/jayl2kor/skillhub@latest
```

## Quick Start

```bash
# Initialize workspace
skillhub init

# Add a registry (local or GitHub)
skillhub repo add ./examples/registry
skillhub repo add https://github.com/org/skill-registry

# Search for skills
skillhub search review

# Install and run a skill
skillhub install code-review
skillhub run code-review

# Install directly to agent's project directory (e.g., Claude Code)
skillhub install code-review --global --tool claude
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize workspace (`~/.skillhub/`) |
| `search [query]` | Search skills across all registries |
| `install <skill>` | Install a skill from a registry |
| `list` | List installed skills |
| `info <skill>` | Show detailed skill information |
| `show manifest <skill>` | Show raw skill.json |
| `show readme <skill>` | Show SKILL.md |
| `show entry <skill>` | Show entry file content |
| `show all <skill>` | Show manifest + readme |
| `run <skill> [args...]` | Run an installed skill |
| `update [skill]` | Update installed skills to latest versions |
| `remove <skill>` | Remove an installed skill |
| `create <name>` | Scaffold a new skill project |
| `lint [dir]` | Validate skill structure |
| `package [dir]` | Build a skill archive (.tar.gz) |
| `pull <skill>` | Download a skill without installing |
| `doctor` | Check workspace health |
| `cache list` | List cached download files |
| `cache clean` | Remove all cached files |
| `repo add <url>` | Add a skill registry |
| `repo list` | List configured registries |
| `verify <archive>` | Verify a skill archive |
| `repo remove <name>` | Remove a registry |
| `completion <shell>` | Generate shell autocompletion (bash/zsh/fish/powershell) |
| `repo index <dir>` | Generate index.json from skill directories and archives |
| `repo update [name]` | Fetch and cache registry indexes |
| `publish [dir]` | Publish a skill to a registry |
| `config list` | Show all configuration values |
| `config get <key>` | Get a configuration value |
| `config set <key> <value>` | Set a configuration value |
| `config path` | Show configuration file path |
| `config edit` | Open configuration in `$EDITOR` |

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--home <path>` | `~/.skillhub` | Workspace directory (or `SKILLHUB_HOME` env) |
| `--verbose` | `false` | Enable verbose output |
| `--version` | - | Print version info |
| `-o, --output` | `table` | Output format for list/search/info (table, json, yaml) |

### Install Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Force reinstall |
| `--global` | `false` | Install to project agent directory |
| `--tool <type>` | `claude` | Agent type for `--global` install |
| `--version <ver>` | - | Install a specific version |

When `--global` is used, the skill is installed to the project-level agent directory.

**Supported agents:**

| Agent | Install Path |
|-------|-------------|
| `claude` | `.claude/skills/<name>` |
| `cursor` | `.cursor/skills/<name>` |
| `windsurf` | `.windsurf/skills/<name>` |
| `cline` | `.cline/skills/<name>` |
| `generic` | `.agent/skills/<name>` |

### Publish Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-r, --repo <name>` | - | Target registry name |
| `--token <token>` | - | GitHub PAT (overrides config) |
| `-f, --force` | `false` | Overwrite existing version |
| `-n, --dry-run` | `false` | Validate without uploading |
| `--version <ver>` | - | Override version from skill.json |
| `--prefix <path>` | `.claude/skills` | Destination prefix in registry |

### Run Flags

| Flag | Description |
|------|-------------|
| `--tool <type>` | Validate agent compatibility (`compatible_agents` in manifest) |

### Registry Authentication

```bash
# Private GitHub registry with PAT
skillhub repo add https://github.com/org/private-registry --token ghp_xxxx

# GitHub Enterprise with Basic Auth
skillhub repo add https://ghe.company.com/org/registry --username user --token token
```

The default branch is auto-detected via GitHub API (falls back to `main`).

## Skill Types

| Type | Runner | Description |
|------|--------|-------------|
| `prompt` | `PromptRunner` | Prints entry file content to stdout |
| `shell` | `ExecRunner` | Executes with `bash` |
| `python` | `ExecRunner` | Executes with `python3` |
| `node` | `ExecRunner` | Executes with `node` |

## Creating Skills

### Package Structure

```
my-skill/
├── skill.json       # Manifest (required)
├── SKILL.md         # Skill readme (required)
├── main.py          # Entry file
└── helpers/          # Additional files (optional)
```

### skill.json

```json
{
  "name": "my-skill",
  "version": "1.0.0",
  "description": "What this skill does",
  "entry": "main.py",
  "type": "python",
  "tags": ["category"],
  "author": "your-name",
  "compatible_agents": ["claude", "cursor"]
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Lowercase alphanumeric with hyphens (`^[a-z0-9]+(-[a-z0-9]+)*$`) |
| `version` | Yes | Semver (`X.Y.Z`) |
| `description` | Yes | Skill description |
| `entry` | Yes | Relative path to entry file |
| `type` | Yes | `prompt`, `shell`, `python`, or `node` |
| `tags` | No | Search tags |
| `author` | No | Author name |
| `homepage` | No | Homepage URL |
| `license` | No | License identifier |
| `compatible_agents` | No | List of compatible agent types |

### Publishing

#### Using `skillhub publish` (recommended)

Publish a skill directory directly to a configured registry:

```bash
# Publish to the default registry
skillhub publish my-skill/

# Preview without uploading
skillhub publish my-skill/ --dry-run

# Publish to a specific registry with a token
skillhub publish my-skill/ -r my-registry --token ghp_xxxx

# Overwrite an existing version
skillhub publish my-skill/ --force

# Custom destination prefix (default: .claude/skills)
skillhub publish my-skill/ --prefix "skills"
```

The destination prefix can also be set per-registry via config:

```bash
skillhub config set registries.my-registry.skills_prefix "custom/path"
```

Resolution order: `--prefix` flag > per-registry `skills_prefix` config > default (`.claude/skills`).

#### Manual publishing

Place your skill folder in the registry and point `index.json` at it:

```
my-registry/
├── index.json
└── skills/
    └── my-skill/
        ├── skill.json
        ├── main.py
        └── SKILL.md
```

Add the entry to your registry's `index.json` (note the trailing `/`):

```json
{
  "skills": [
    {
      "name": "my-skill",
      "version": "1.0.0",
      "description": "What this skill does",
      "tags": ["category"],
      "download_url": "skills/my-skill/"
    }
  ]
}
```

Generate `index.json` automatically from skill directories:

```bash
skillhub repo index skills/
```

#### Archive mode (alternative)

If you prefer distributable archives, you can package skills as `.tar.gz`:

```bash
skillhub package my-skill/
# or manually:
tar czf my-skill-1.0.0.tar.gz my-skill/
```

```json
{
  "name": "my-skill",
  "version": "1.0.0",
  "download_url": "packages/my-skill-1.0.0.tar.gz",
  "checksum": "sha256:abc123..."
}
```

Both formats can coexist in the same `index.json`.

## Registry Guide

A registry is a directory (local or Git-hosted) containing an `index.json` and skill directories (or archives).

```
my-registry/
├── index.json
└── skills/
    ├── code-review/
    │   ├── skill.json
    │   └── prompt.md
    └── lint-fix/
        ├── skill.json
        └── main.sh
```

**Supported URL formats:**

| Format | Example |
|--------|---------|
| GitHub URL | `https://github.com/org/repo` |
| Shorthand | `org/repo` |
| Local path | `./path/to/registry` |

Multiple registries can be configured. When searching or installing, all registries are queried and merged. Unreachable registries are skipped with a warning.

## Configuration

```
~/.skillhub/
├── config.yaml
├── skills/
├── cache/
├── logs/
└── tmp/
```

### config.yaml

```yaml
registries:
  - name: my-registry
    url: https://github.com/org/skill-registry
    token: ghp_xxxx          # optional
    username: user            # optional (GitHub Enterprise)
    branch: main              # auto-detected
    skills_prefix: .claude/skills  # optional, publish destination prefix
install_dir: ~/.skillhub/skills
cache_dir: ~/.skillhub/cache
log_dir: ~/.skillhub/logs
```

Config files with tokens are saved with `0600` permissions.

### Managing config via CLI

```bash
# View all settings (tokens masked by default)
skillhub config list

# Get a specific value using dot notation
skillhub config get install_dir
skillhub config get registries.my-registry.skills_prefix

# Set a value
skillhub config set registries.my-registry.skills_prefix "custom/path"

# Show raw token value
skillhub config get registries.my-registry.token --unmask

# Open config in editor
skillhub config edit
```

Dot-notation keys for registries: `registries.<name>.<field>` where field is one of `name`, `url`, `token`, `username`, `branch`, `skills_prefix`. Numeric index access (`registries.0.url`) is also supported.

## Security

- **Archive extraction**: Path traversal prevention, absolute path rejection, symlink blocking, 100MB per-file limit (archive mode)
- **Checksum verification**: SHA256 validation when `checksum` is provided in `index.json` (archive mode)
- **Entry path validation**: Manifest entry paths cannot escape the skill directory
- **Credential protection**: Config file stored with `0600` permissions
- **Download limits**: 500MB max archive size, 10MB max API response size

## Development

```bash
# Run tests
go test ./...

# Lint
go vet ./...
```

## License

MIT
