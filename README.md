# skillhub

A lightweight CLI package manager for AI agent skills.

Search, install, run, update, and remove skills from registries with a single command.

## Installation

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
| `pull <skill>` | Download skill archive without installing |
| `doctor` | Check workspace health |
| `cache list` | List cached download files |
| `cache clean` | Remove all cached files |
| `repo add <url>` | Add a skill registry |
| `repo list` | List configured registries |
| `verify <archive>` | Verify a skill archive |
| `repo remove <name>` | Remove a registry |
| `completion <shell>` | Generate shell autocompletion (bash/zsh/fish/powershell) |
| `repo index <dir>` | Generate index.json from skill archives |
| `repo update [name]` | Fetch and cache registry indexes |

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

When `--global` is used, the skill is installed to the project-level agent directory and requires a `SKILL.md` file in the package.

**Supported agents:**

| Agent | Install Path |
|-------|-------------|
| `claude` | `.claude/skills/<name>` |
| `cursor` | `.cursor/skills/<name>` |
| `windsurf` | `.windsurf/skills/<name>` |
| `cline` | `.cline/skills/<name>` |
| `generic` | `.agent/skills/<name>` |

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
├── SKILL.md         # Required for --global install
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

### Packaging and Publishing

```bash
# Create archive
tar czf my-skill-1.0.0.tar.gz my-skill/

# Generate checksum (optional)
shasum -a 256 packages/my-skill-1.0.0.tar.gz
```

Add the entry to your registry's `index.json`:

```json
{
  "skills": [
    {
      "name": "my-skill",
      "version": "1.0.0",
      "description": "What this skill does",
      "tags": ["category"],
      "download_url": "packages/my-skill-1.0.0.tar.gz",
      "checksum": "sha256:abc123..."
    }
  ]
}
```

## Registry Guide

A registry is a directory (local or Git-hosted) containing an `index.json` and skill archives.

```
my-registry/
├── index.json
└── packages/
    └── my-skill-1.0.0.tar.gz
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
install_dir: ~/.skillhub/skills
cache_dir: ~/.skillhub/cache
log_dir: ~/.skillhub/logs
```

Config files with tokens are saved with `0600` permissions.

## Security

- **Archive extraction**: Path traversal prevention, absolute path rejection, symlink blocking, 100MB per-file limit
- **Checksum verification**: SHA256 validation when `checksum` is provided in `index.json`
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
