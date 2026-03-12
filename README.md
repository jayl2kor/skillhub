# skillhub

AI/에이전트 스킬을 위한 경량 CLI 패키지 매니저.

레지스트리에서 스킬을 검색하고, 설치·실행·업데이트·제거하는 전체 라이프사이클을 관리한다.

**기술 스택:** Go 1.25.3 · [cobra](https://github.com/spf13/cobra) · [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)

---

## 목차

1. [설치 방법](#설치-방법)
2. [빠른 시작](#빠른-시작)
3. [명령어 레퍼런스](#명령어-레퍼런스)
4. [설정](#설정)
5. [레지스트리 가이드](#레지스트리-가이드)
6. [스킬 제작 가이드](#스킬-제작-가이드)
7. [데이터 구조 레퍼런스](#데이터-구조-레퍼런스)
8. [아키텍처](#아키텍처)
9. [보안](#보안)
10. [개발 가이드](#개발-가이드)
11. [향후 계획](#향후-계획)

---

## 설치 방법

### 소스 빌드

```bash
make build
./bin/skillhub --version
```

`./bin/skillhub` 바이너리가 생성된다.

### 크로스 플랫폼 빌드

```bash
make build-all
```

`dist/` 디렉토리에 5개 바이너리가 생성된다:

| 파일 | OS | Arch |
|---|---|---|
| `skillhub-linux-amd64` | Linux | amd64 |
| `skillhub-linux-arm64` | Linux | arm64 |
| `skillhub-darwin-amd64` | macOS | amd64 |
| `skillhub-darwin-arm64` | macOS | arm64 |
| `skillhub-windows-amd64.exe` | Windows | amd64 |

### 버전 정보 주입

빌드 시 `-ldflags`로 버전 정보가 자동 주입된다:

```makefile
-X skillhub/pkg/version.Version=$(VERSION)      # git describe 또는 "dev"
-X skillhub/pkg/version.BuildDate=$(BUILD_DATE)  # UTC 빌드 시각
-X skillhub/pkg/version.GitCommit=$(GIT_COMMIT)  # git 커밋 해시 (short)
```

---

## 빠른 시작

```bash
# 1. 워크스페이스 초기화
skillhub init

# 2. 레지스트리 등록 (로컬 또는 GitHub)
skillhub repo add ./examples/registry
skillhub repo add https://github.com/org/skill-registry

# 3. 스킬 검색
skillhub search review
# NAME          VERSION   DESCRIPTION
# code-review   1.0.0     Review code changes and produce structured feedback

# 4. 스킬 설치
skillhub install code-review
# Installed code-review@1.0.0 from registry

# 5. 설치된 스킬 목록 확인
skillhub list
# NAME          VERSION   TYPE
# code-review   1.0.0     prompt

# 6. 스킬 상세 정보 확인
skillhub info code-review
# Name:        code-review
# Version:     1.0.0
# Description: Review code changes and produce structured feedback
# Type:        prompt
# Entry:       prompt.md
# Tags:        engineering, review
# Author:      platform-team
# Installed:   yes
# Registry:    registry

# 7. 스킬 실행
skillhub run code-review

# 8. 스킬 업데이트 (전체 또는 지정)
skillhub update
skillhub update code-review

# 9. 스킬 제거
skillhub remove code-review
# Removed code-review

# 10. 환경 진단
skillhub doctor
# Config file... OK
# Skills directory... OK
# Cache directory... OK
# Log directory... OK
# Cache writable... OK
# Installed skills... 0 installed, all valid
# Registries... 1 configured
#
# All checks passed.
```

---

## 명령어 레퍼런스

### 글로벌 플래그

| 플래그 | 타입 | 기본값 | 설명 |
|---|---|---|---|
| `--home <path>` | string | `~/.skillhub` | 홈 디렉토리 경로 |
| `--verbose` | bool | `false` | 상세 출력 활성화 |
| `--version` | bool | - | 버전 정보 출력 |

`--home`을 지정하지 않으면 `SKILLHUB_HOME` 환경변수 → `~/.skillhub` → `./.skillhub` 순으로 결정된다.

---

### `init`

워크스페이스를 초기화한다. 디렉토리 구조와 기본 설정 파일을 생성한다.

```bash
skillhub init
```

**동작:**
1. `~/.skillhub/` 하위 디렉토리 생성 (`skills/`, `cache/`, `logs/`, `tmp/`)
2. `config.yaml`이 없으면 기본 설정 파일 생성
3. 이미 존재하면 건너뜀 (멱등성 보장)

**출력:**
```
Created workspace directories at /Users/user/.skillhub
Created default config at /Users/user/.skillhub/config.yaml
Workspace initialized successfully.
```

---

### `repo add`

스킬 레지스트리를 등록한다.

```bash
skillhub repo add <url>
```

**지원 형식:**

| 형식 | 예시 | 변환 결과 |
|---|---|---|
| GitHub URL | `https://github.com/org/repo` | 그대로 사용 |
| GitHub URL (.git) | `https://github.com/org/repo.git` | `.git` 제거 |
| 축약형 | `org/repo` | `https://github.com/org/repo` |
| 로컬 경로 | `./examples/registry` | 그대로 사용 |

등록 시 `index.json` 접근 가능 여부를 검증한다. 접근 불가 시 오류를 반환한다.

**출력:**
```
Added registry "registry" (./examples/registry)
```

---

### `repo list`

등록된 레지스트리 목록을 출력한다.

```bash
skillhub repo list
```

**출력:**
```
NAME       URL
registry   ./examples/registry
```

레지스트리가 없으면 `No registries configured.`를 출력한다.

---

### `repo remove`

등록된 레지스트리를 제거한다.

```bash
skillhub repo remove <name>
```

**출력:**
```
Removed registry "registry"
```

---

### `search`

등록된 모든 레지스트리에서 스킬을 검색한다. 이름, 설명, 태그에서 대소문자 무관 부분 일치 검색을 수행한다.

```bash
skillhub search <query>
```

**출력:**
```
NAME          VERSION   DESCRIPTION
code-review   1.0.0     Review code changes and produce structured feedback
```

결과가 없으면 `No skills found.`를 출력한다.

---

### `install`

레지스트리에서 스킬을 다운로드하여 설치한다.

```bash
skillhub install <skill>
skillhub install <skill> --force   # 이미 설치된 경우 재설치
```

| 플래그 | 설명 |
|---|---|
| `--force` | 이미 설치된 스킬을 강제 재설치 |

**동작:**
1. 중복 설치 확인 (이미 설치 시 오류, `--force`로 우회)
2. 모든 레지스트리에서 인덱스 조회 및 병합
3. 스킬 아카이브 다운로드 → 체크섬 검증 → 임시 디렉토리 추출
4. 매니페스트 검증 → 최종 위치로 이동 → 설치 메타데이터 기록

**출력:**
```
Installed code-review@1.0.0 from registry
```

---

### `list`

설치된 스킬 목록을 테이블 형식으로 출력한다.

```bash
skillhub list
```

**출력:**
```
NAME          VERSION   TYPE
code-review   1.0.0     prompt
```

설치된 스킬이 없으면 `No skills installed.`를 출력한다.

---

### `info`

스킬의 상세 정보를 출력한다. 로컬 설치 여부와 무관하게 조회 가능하다.

```bash
skillhub info <skill>
```

**설치된 스킬 출력:**
```
Name:        code-review
Version:     1.0.0
Description: Review code changes and produce structured feedback
Type:        prompt
Entry:       prompt.md
Tags:        engineering, review
Author:      platform-team
Installed:   yes
Registry:    registry
Installed at: 2026-03-12T12:00:00Z
```

**미설치 스킬 출력 (레지스트리 조회):**
```
Name:        code-review
Version:     1.0.0
Description: Review code changes and produce structured feedback
Tags:        engineering, review
Registry:    registry
Installed:   no
```

---

### `run`

설치된 스킬을 실행한다. 현재는 `prompt` 타입만 지원하며, 엔트리 파일 내용을 stdout으로 출력한다.

```bash
skillhub run <skill> [args...]
```

**동작:**
1. 스킬 설치 여부 확인
2. 스킬 타입에 맞는 러너 선택 (`prompt` → `PromptRunner`)
3. 엔트리 파일 경로 보안 검증 후 내용 출력

---

### `update`

설치된 스킬을 레지스트리의 최신 버전으로 업데이트한다.

```bash
skillhub update           # 모든 설치된 스킬 업데이트
skillhub update <skill>   # 특정 스킬만 업데이트
```

**동작:**
1. 레지스트리에서 최신 인덱스 조회
2. 로컬 버전과 레지스트리 버전을 시맨틱 버전으로 비교
3. 새 버전이 있으면 `--force` 재설치 수행

**출력 예시:**
```
code-review: updating 1.0.0 -> 1.1.0
Installed code-review@1.1.0 from registry
Updated 1 skill(s).
```

이미 최신이면:
```
code-review: already at latest version (1.0.0)
All skills are up to date.
```

---

### `remove`

설치된 스킬을 제거한다.

```bash
skillhub remove <skill>
```

스킬 이름에 경로 구분자(`/`, `\`)가 포함되면 거부한다.

**출력:**
```
Removed code-review
```

---

### `doctor`

로컬 환경 상태를 진단한다.

```bash
skillhub doctor
```

**검사 항목:**

| 항목 | 설명 |
|---|---|
| Config file | 설정 파일 존재 및 유효성 검사 |
| Skills directory | 스킬 디렉토리 존재 여부 |
| Cache directory | 캐시 디렉토리 존재 여부 |
| Log directory | 로그 디렉토리 존재 여부 |
| Cache writable | 캐시 디렉토리 쓰기 권한 |
| Installed skills | 설치된 스킬 매니페스트 유효성 |
| Registries | 등록된 레지스트리 수 |

**정상 출력:**
```
Config file... OK
Skills directory... OK
Cache directory... OK
Log directory... OK
Cache writable... OK
Installed skills... 2 installed, all valid
Registries... 1 configured

All checks passed.
```

**오류 출력:**
```
Config file... MISSING
Skills directory... OK
...
some checks failed; run 'skillhub init' to fix
```

---

## 설정

### 디렉토리 구조

```
~/.skillhub/
├── config.yaml          # 설정 파일
├── skills/              # 설치된 스킬
│   └── code-review/
│       ├── skill.json
│       ├── prompt.md
│       └── .install.json
├── cache/               # 다운로드 캐시
│   └── code-review-1.0.0.tar.gz
├── logs/                # 로그
└── tmp/                 # 임시 파일 (설치 중 사용)
```

### 홈 디렉토리 결정 순서

1. `--home` 플래그가 지정되면 해당 경로 사용
2. `SKILLHUB_HOME` 환경변수가 설정되면 해당 경로 사용
3. `os.UserHomeDir()` 성공 시 `~/.skillhub`
4. 모두 실패 시 `./.skillhub` (현재 디렉토리)

### config.yaml

| 필드 | 타입 | 설명 |
|---|---|---|
| `registries` | `[]RegistryEntry` | 등록된 레지스트리 목록 |
| `registries[].name` | `string` | 레지스트리 식별자 |
| `registries[].url` | `string` | GitHub URL 또는 로컬 경로 |
| `install_dir` | `string` | 스킬 설치 디렉토리 |
| `cache_dir` | `string` | 다운로드 캐시 디렉토리 |
| `log_dir` | `string` | 로그 디렉토리 |

**기본 설정 (`init` 직후):**

```yaml
registries: []
install_dir: /Users/user/.skillhub/skills
cache_dir: /Users/user/.skillhub/cache
log_dir: /Users/user/.skillhub/logs
```

**로컬 레지스트리 1개 등록 상태:**

```yaml
registries:
    - name: registry
      url: ./examples/registry
install_dir: /Users/user/.skillhub/skills
cache_dir: /Users/user/.skillhub/cache
log_dir: /Users/user/.skillhub/logs
```

**GitHub + 로컬 다중 레지스트리 구성:**

```yaml
registries:
    - name: my-skills
      url: https://github.com/org/skill-registry
    - name: local-dev
      url: ./my-local-registry
install_dir: /Users/user/.skillhub/skills
cache_dir: /Users/user/.skillhub/cache
log_dir: /Users/user/.skillhub/logs
```

### .install.json

각 설치된 스킬 디렉토리에 자동 생성되는 메타데이터 파일.

```json
{
  "installed_at": "2026-03-12T12:00:00Z",
  "registry": "my-skills",
  "version": "1.0.0",
  "checksum": "sha256:abc123..."
}
```

| 필드 | 설명 |
|---|---|
| `installed_at` | 설치 시각 (RFC 3339 형식, UTC) |
| `registry` | 설치 출처 레지스트리 이름 |
| `version` | 설치된 버전 |
| `checksum` | 아카이브 체크섬 (`sha256:` 접두사, 선택) |

---

## 레지스트리 가이드

레지스트리는 스킬 패키지를 배포하는 저장소이다. GitHub 저장소 또는 로컬 디렉토리를 사용할 수 있다.

### 저장소 구조

```
my-registry/
├── index.json                        # 스킬 카탈로그 (필수)
└── packages/
    ├── code-review-1.0.0.tar.gz      # 스킬 패키지
    └── test-helper-2.1.0.tar.gz
```

### index.json 스키마

```json
{
  "skills": [
    {
      "name": "code-review",
      "version": "1.0.0",
      "description": "Review code changes and produce structured feedback",
      "tags": ["engineering", "review"],
      "download_url": "packages/code-review-1.0.0.tar.gz",
      "checksum": "sha256:abc123..."
    }
  ]
}
```

| 필드 | 필수 | 설명 |
|---|---|---|
| `name` | O | 스킬 이름 |
| `version` | O | 시맨틱 버전 (X.Y.Z) |
| `description` | O | 스킬 설명 |
| `tags` | - | 검색용 태그 배열 |
| `download_url` | O | 패키지 다운로드 경로 (상대 또는 절대 URL) |
| `checksum` | - | SHA256 체크섬 (`sha256:` 접두사) |

### GitHub URL 변환 규칙

GitHub 레지스트리의 `download_url`이 상대 경로인 경우, 다음과 같이 변환된다:

```
레지스트리 URL:  https://github.com/org/repo
download_url:   packages/code-review-1.0.0.tar.gz
                ↓
실제 요청 URL:  https://raw.githubusercontent.com/org/repo/main/packages/code-review-1.0.0.tar.gz
```

`download_url`이 `http://` 또는 `https://`로 시작하면 절대 URL로 간주하여 변환하지 않는다.

### 로컬 레지스트리

개발·테스트 시 로컬 디렉토리를 레지스트리로 사용할 수 있다.

```bash
skillhub repo add ./my-local-registry
skillhub repo add /absolute/path/to/registry
skillhub repo add ../relative/path
```

로컬 레지스트리에서는 `download_url`이 `index.json` 기준 상대 경로로 해석된다.

### 다중 레지스트리

여러 레지스트리를 등록할 수 있다. `search`, `install`, `update` 시 모든 레지스트리의 인덱스를 조회하여 병합한다. 동일 이름의 스킬이 여러 레지스트리에 있으면 먼저 발견된 것이 사용된다.

접근 불가한 레지스트리는 경고를 출력하고 건너뛴다:

```
warning: failed to fetch from broken-registry: ...
```

---

## 스킬 제작 가이드

### 패키지 구조

스킬 패키지는 `.tar.gz` 아카이브로, 최상위 또는 1단계 하위 디렉토리에 `skill.json`이 있어야 한다.

```
code-review/
├── skill.json       # 매니페스트 (필수)
├── prompt.md        # 엔트리 파일
└── helpers/         # 추가 파일 (선택)
    └── utils.md
```

### skill.json 매니페스트

```json
{
  "name": "code-review",
  "version": "1.0.0",
  "description": "Review code changes and produce structured feedback",
  "entry": "prompt.md",
  "type": "prompt",
  "tags": ["engineering", "review"],
  "author": "platform-team",
  "homepage": "https://github.com/org/repo",
  "license": "MIT"
}
```

| 필드 | 필수 | 타입 | 설명 |
|---|---|---|---|
| `name` | O | string | 스킬 이름 (소문자 영숫자 + 하이픈) |
| `version` | O | string | 시맨틱 버전 (X.Y.Z) |
| `description` | O | string | 스킬 설명 |
| `entry` | O | string | 엔트리 파일 상대 경로 |
| `type` | O | string | 스킬 타입 |
| `tags` | - | []string | 검색용 태그 |
| `author` | - | string | 작성자 |
| `homepage` | - | string | 홈페이지 URL |
| `license` | - | string | 라이선스 |

### 유효성 규칙

| 필드 | 규칙 |
|---|---|
| `name` | 정규식 `^[a-z0-9]+(-[a-z0-9]+)*$` (예: `code-review`, `my-skill-v2`) |
| `version` | 정규식 `^\d+\.\d+\.\d+$` (예: `1.0.0`, `12.3.45`) |
| `entry` | 상대 경로만 허용, `..`으로 시작 불가, 절대 경로 불가 |
| `type` | `prompt`, `shell`, `python`, `node` 중 하나 |

### 지원 스킬 타입

| 타입 | 상태 | 설명 |
|---|---|---|
| `prompt` | 구현됨 | 엔트리 파일 내용을 stdout으로 출력 |
| `shell` | 계획 | 셸 스크립트 실행 |
| `python` | 계획 | Python 런타임 실행 |
| `node` | 계획 | Node.js 런타임 실행 |

### 패키징

```bash
# 스킬 디렉토리 구조 준비
mkdir -p code-review
# skill.json, prompt.md 등 파일 작성

# tar.gz 아카이브 생성
tar czf code-review-1.0.0.tar.gz code-review/
```

### index.json 등록

패키지를 레지스트리의 `packages/` 디렉토리에 배치하고, `index.json`에 항목을 추가한다:

```bash
# 체크섬 생성 (선택)
shasum -a 256 packages/code-review-1.0.0.tar.gz
# abc123...  packages/code-review-1.0.0.tar.gz
```

```json
{
  "skills": [
    {
      "name": "code-review",
      "version": "1.0.0",
      "description": "Review code changes and produce structured feedback",
      "tags": ["engineering", "review"],
      "download_url": "packages/code-review-1.0.0.tar.gz",
      "checksum": "sha256:abc123..."
    }
  ]
}
```

---

## 데이터 구조 레퍼런스

프로젝트의 핵심 Go 타입 정의와 타입 간 관계를 정리한다.

### Manifest (skill.json)

스킬 패키지의 매니페스트. 설치된 스킬 디렉토리의 `skill.json`에 대응한다.

```go
type Manifest struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Description string   `json:"description"`
    Entry       string   `json:"entry"`
    Type        string   `json:"type"`
    Tags        []string `json:"tags,omitempty"`
    Author      string   `json:"author,omitempty"`
    Homepage    string   `json:"homepage,omitempty"`
    License     string   `json:"license,omitempty"`
}
```

`Validate()` 메서드로 `name`, `version`, `entry`, `type` 필드의 유효성을 검증한다.

### IndexEntry (index.json 내 각 스킬)

레지스트리 인덱스의 개별 스킬 항목. `Manifest`와 유사하지만 `download_url`, `checksum` 필드가 추가되고 `entry`, `type` 등은 없다.

```go
type IndexEntry struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Description string   `json:"description"`
    Tags        []string `json:"tags,omitempty"`
    DownloadURL string   `json:"download_url"`
    Checksum    string   `json:"checksum,omitempty"`
    Registry    string   `json:"-"`  // JSON 직렬화 제외, 런타임에 설정
}
```

### Index (index.json 전체)

레지스트리의 스킬 카탈로그.

```go
type Index struct {
    Skills []IndexEntry `json:"skills"`
}
```

`Search(query)` — 이름, 설명, 태그에서 대소문자 무관 부분 일치 검색
`Find(name)` — 이름 완전 일치로 단일 스킬 조회
`MergeIndexes(...)` — 여러 인덱스를 하나로 병합

### Config / RegistryEntry (config.yaml)

```go
type RegistryEntry struct {
    Name string `yaml:"name"`
    URL  string `yaml:"url"`
}

type Config struct {
    Registries []RegistryEntry `yaml:"registries"`
    InstallDir string          `yaml:"install_dir"`
    CacheDir   string          `yaml:"cache_dir"`
    LogDir     string          `yaml:"log_dir"`
}
```

`DefaultConfig(home)` — 빈 레지스트리 목록과 기본 경로로 초기화
`AddRegistry(name, url)` — 중복 이름 검사 후 추가
`RemoveRegistry(name)` — 이름으로 제거

### InstalledSkill / InstallMeta (.install.json)

```go
type InstalledSkill struct {
    Manifest Manifest
    Dir      string       // 스킬 디렉토리 절대 경로
    Meta     InstallMeta
}

type InstallMeta struct {
    InstalledAt string `json:"installed_at"`  // RFC 3339 UTC
    Registry    string `json:"registry"`
    Version     string `json:"version"`
    Checksum    string `json:"checksum,omitempty"`
}
```

### RepoSource

레지스트리 소스 정보. `Config.Registries`의 각 항목이 `RepoSource`로 변환된다.

```go
type RepoSource struct {
    Name string
    URL  string
}
```

`IndexURL()` — 인덱스 URL 생성 (로컬: `URL/index.json`, GitHub: raw URL 변환)
`ResolveDownloadURL(relative)` — 상대 경로를 실제 다운로드 URL로 변환
`ParseRepoURL(rawURL)` — URL 문자열을 파싱하여 `RepoSource` 생성 (축약형, GitHub URL, 로컬 경로 지원)

### Paths

홈 디렉토리 기반 경로 구조체.

```go
type Paths struct {
    Home      string  // 기본: ~/.skillhub
    Config    string  // ~/.skillhub/config.yaml
    SkillsDir string  // ~/.skillhub/skills
    CacheDir  string  // ~/.skillhub/cache
    LogDir    string  // ~/.skillhub/logs
    TmpDir    string  // ~/.skillhub/tmp
}
```

`NewPaths(home)` — 홈 경로 기준으로 모든 하위 경로 생성
`DefaultHome()` — `SKILLHUB_HOME` → `~/.skillhub` → `./.skillhub` 순으로 결정
`EnsureDirectories()` — 모든 디렉토리 생성 (`MkdirAll`)

### SkillRunner 인터페이스

스킬 실행을 추상화하는 인터페이스.

```go
type SkillRunner interface {
    Run(ctx context.Context, s skill.InstalledSkill, args []string) error
}
```

`RunnerFor(skillType)` — 스킬 타입에 맞는 러너 반환 (현재 `prompt`만 구현)

`PromptRunner`는 엔트리 파일을 읽어 stdout으로 출력한다. 실행 전 경로 탈출 검증을 수행한다.

### SemVer

시맨틱 버전 파싱 및 비교.

```go
type SemVer struct {
    Major int
    Minor int
    Patch int
}
```

`ParseVersion(s)` — `"X.Y.Z"` 문자열을 `SemVer`로 파싱
`CompareVersions(a, b)` — 두 버전 비교 (`-1`: a < b, `0`: 동일, `1`: a > b)
`String()` — `"X.Y.Z"` 형식으로 직렬화

### 타입 간 관계

```
config.yaml                 index.json              skill.json
┌──────────────────┐       ┌───────────────┐       ┌──────────────┐
│ Config           │       │ Index         │       │ Manifest     │
│  .Registries[] ──┼──→    │  .Skills[]    │       │  .Name       │
│    RegistryEntry │  변환  │    IndexEntry ├──→    │  .Version    │
│    → RepoSource  │       │    (원격 메타) │  설치  │  (로컬 메타)  │
└──────────────────┘       └───────────────┘       └──────────────┘
                                                          │
                                                          ↓
                                                   ┌──────────────┐
                                                   │InstalledSkill│
                                                   │  .Manifest   │
                                                   │  .Dir        │
                                                   │  .Meta       │
                                                   │  (InstallMeta)│
                                                   └──────────────┘
```

- `Config.Registries[]` → `RepoSource`로 변환되어 인덱스 조회에 사용
- `IndexEntry` → 설치 시 패키지를 다운로드하여 `Manifest`가 포함된 스킬 디렉토리 생성
- `InstalledSkill`은 `Manifest` + 디렉토리 경로 + `InstallMeta`를 결합

---

## 아키텍처

### 고수준 흐름

```
┌─────────┐     ┌──────────┐     ┌───────────┐     ┌──────────┐
│  User    │────→│  CLI     │────→│  Registry │────→│  HTTP /  │
│          │     │ (cobra)  │     │  Client   │     │  Local   │
└─────────┘     └──────────┘     └───────────┘     └──────────┘
                     │                                    │
                     │           ┌───────────┐           │
                     ├──────────→│ Installer │←──────────┘
                     │           │ (다운로드,  │
                     │           │  검증, 추출)│
                     │           └───────────┘
                     │                │
                     │           ┌───────────┐
                     │           │  Storage  │
                     │           │ (파일시스템)│
                     │           └───────────┘
                     │
                     │           ┌───────────┐
                     └──────────→│  Runtime  │
                                 │ (스킬 실행)│
                                 └───────────┘
```

### 프로젝트 디렉토리 구조

```
skillhub/
├── cmd/skillhub/
│   └── main.go                  # 진입점 → cli.Execute()
├── internal/
│   ├── cli/                     # Cobra 명령어 정의 (11개 명령)
│   ├── config/                  # 설정 파일 로드/저장/검증
│   ├── skill/                   # Manifest, InstalledSkill, SemVer
│   ├── storage/                 # 파일시스템 경로 관리, 스킬 조회
│   ├── registry/                # Index 파싱, HTTP 클라이언트, URL 변환
│   ├── installer/               # 설치 흐름, 아카이브 추출, 체크섬 검증
│   ├── runtime/                 # SkillRunner 인터페이스, PromptRunner
│   └── integration_test.go      # E2E 통합 테스트
├── pkg/version/                 # 버전 정보 (ldflags 주입)
├── examples/registry/           # 예제 레지스트리
├── docs/                        # 개발 계획서, TODO
└── Makefile                     # 빌드/테스트/린트 타겟
```

### 설치 흐름 상세 (12단계)

1. **중복 확인** — `storage.IsInstalled()`로 이미 설치 여부 확인 (`--force` 시 건너뜀)
2. **소스 구성** — `Config.Registries`를 `[]RepoSource`로 변환
3. **인덱스 조회** — 모든 레지스트리에서 `index.json` 가져와 병합 (실패 레지스트리는 경고 후 건너뜀)
4. **스킬 검색** — 병합된 인덱스에서 이름 완전 일치 검색
5. **URL 해석** — `RepoSource.ResolveDownloadURL()`로 실제 다운로드 URL 결정
6. **다운로드** — 아카이브를 `~/.skillhub/cache/{name}-{version}.tar.gz`에 저장
7. **체크섬 검증** — `checksum` 필드가 있으면 SHA256 해시 비교 (불일치 시 캐시 파일 삭제)
8. **추출** — `~/.skillhub/tmp/{name}-{random}/`에 tar.gz 추출 (보안 검증 포함)
9. **매니페스트 탐색** — 루트 또는 1단계 하위에서 `skill.json` 탐색
10. **매니페스트 검증** — `LoadManifest()` → `Validate()` 호출
11. **배치** — `os.Rename()`으로 최종 위치(`~/.skillhub/skills/{name}/`)에 원자적 이동
12. **메타데이터** — `.install.json`에 설치 시각, 레지스트리, 버전, 체크섬 기록

---

## 보안

### 아카이브 추출 보안

`ExtractTarGz()`에서 다음 보안 검증을 수행한다:

| 방어 항목 | 구현 |
|---|---|
| 경로 탈출 방어 | `filepath.Rel()`로 대상 디렉토리 내부 경로인지 검증 |
| `..` 경로 거부 | `..` 접두사 및 `/..` 포함 경로 차단 |
| 절대 경로 거부 | `filepath.IsAbs()` 검사 |
| 심볼릭 링크 거부 | `tar.TypeSymlink`, `tar.TypeLink` 타입 차단 |
| 파일 크기 제한 | 단일 파일 100MB 초과 시 거부 |

```go
func sanitizePath(name string, destDir string) (string, error) {
    cleaned := filepath.Clean(name)
    if filepath.IsAbs(cleaned) { return "", error }       // 절대 경로 거부
    if strings.HasPrefix(cleaned, "..") { return "", error } // 상위 탈출 거부
    target := filepath.Join(destDir, cleaned)
    rel, _ := filepath.Rel(destDir, target)
    if strings.HasPrefix(rel, "..") { return "", error }   // 최종 경로 검증
    return cleaned, nil
}
```

### SHA256 체크섬 검증

`index.json`에 `checksum` 필드가 있으면 다운로드된 아카이브의 SHA256 해시를 비교한다. 불일치 시 캐시 파일을 삭제하고 설치를 중단한다.

```go
func VerifyChecksum(filePath string, expected string) error
func ComputeSHA256(filePath string) (string, error)
```

### 매니페스트 엔트리 경로 검증

`Manifest.Validate()`에서 `entry` 필드가:
- 절대 경로가 아닌지 (`filepath.IsAbs`)
- `..`으로 시작하지 않는지 (`filepath.Clean` 후 검사)

### 런타임 경로 검증

`PromptRunner.Run()`에서 엔트리 파일 실행 전:
- 경로 정규화 (`filepath.Clean`)
- 절대 경로 및 `..` 접두사 거부
- `filepath.Rel()`로 스킬 디렉토리 내부인지 최종 검증

### 스킬 이름 검증

`remove` 명령에서 경로 구분자(`/`, `\`) 포함 여부를 검사하여 디렉토리 트래버설을 방지한다.

---

## 개발 가이드

### Makefile 타겟

| 타겟 | 명령어 | 설명 |
|---|---|---|
| `build` | `go build -ldflags ... -o bin/skillhub` | 현재 플랫폼 바이너리 빌드 |
| `test` | `go test ./... -v` | 전체 테스트 실행 |
| `lint` | `go vet ./...` | 정적 분석 |
| `clean` | `rm -rf bin/ dist/` | 빌드 산출물 제거 |
| `build-all` | 5개 플랫폼 교차 빌드 | 크로스 플랫폼 바이너리 생성 |

### 테스트

12개 테스트 파일, 50개 테스트 함수:

| 패키지 | 테스트 파일 | 주요 검증 대상 |
|---|---|---|
| `skill` | `manifest_test.go` | 매니페스트 로드, 이름/버전/엔트리/타입 유효성 |
| `skill` | `version_test.go` | SemVer 파싱, 버전 비교 |
| `config` | `config_test.go` | 설정 로드/저장, 유효성, 레지스트리 추가/제거 |
| `storage` | `paths_test.go` | 경로 생성, 디렉토리 보장, 기본 홈 |
| `registry` | `index_test.go` | 인덱스 파싱, 검색, 찾기, 병합 |
| `registry` | `client_test.go` | 로컬/HTTP 인덱스 조회, 다운로드 |
| `installer` | `install_test.go` | 설치 흐름, 중복 설치, 미발견, 레지스트리 없음 |
| `installer` | `extract_test.go` | 아카이브 추출, 경로 탈출/절대 경로/심볼릭 링크 방어 |
| `installer` | `verify_test.go` | 체크섬 일치/불일치/빈값 |
| `runtime` | `prompt_test.go` | 프롬프트 출력, 파일 누락, 경로 탈출 |
| `cli` | `commands_test.go` | 루트 명령, 서브커맨드 등록, repo 서브커맨드 |
| (root) | `integration_test.go` | 전체 워크플로우 E2E 테스트 |

### 의존성

| 패키지 | 버전 | 용도 |
|---|---|---|
| `github.com/spf13/cobra` | v1.9.1 | CLI 프레임워크 |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML 설정 파싱 |
| `github.com/inconshreveable/mousetrap` | v1.1.0 | cobra 간접 의존 (Windows) |
| `github.com/spf13/pflag` | v1.0.6 | cobra 간접 의존 (플래그 파싱) |

---

## 향후 계획

- **런타임 확장** — `shell`, `python`, `node` 스킬 타입 실행 지원
- **TUI** — [bubbletea](https://github.com/charmbracelet/bubbletea) 기반 터미널 UI
- **서명 검증** — 패키지 서명 및 검증 메커니즘
- **샌드박스 실행** — 스킬 실행 환경 격리
- **레지스트리 publish** — `skillhub publish` 명령으로 레지스트리에 스킬 배포 워크플로우
