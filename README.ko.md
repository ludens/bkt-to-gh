# bkt2gh

Bitbucket Cloud 저장소를 GitHub로 옮기는 Go CLI입니다.

[English](README.md)

선택한 Bitbucket 저장소를 GitHub에 새 저장소로 만들고, `git clone --mirror`와 `git push --mirror`로 브랜치와 태그를 포함한 Git 이력을 이전합니다. 실행 전 preview로 대상 저장소, GitHub 생성 가능 여부, 공개 범위 정책을 확인할 수 있습니다.

## 주요 기능

- Bitbucket Cloud workspace의 저장소 목록 조회
- 터미널에서 이전할 저장소 선택
- GitHub 저장소 생성
- mirror clone/push 기반 Git 이력 이전
- `git-lfs`가 설치된 경우 LFS 객체 fetch/push 시도
- GitHub 저장소 공개 범위 정책 선택
- migration preview 사전 점검
- OS 사용자 config 경로의 암호화된 설정과 환경변수 override

## 요구 사항

- Git
- Bitbucket Cloud 계정과 app password
- GitHub token
- 선택: Git LFS 저장소를 옮기려면 `git-lfs`

## 설치

Homebrew:

```bash
brew tap ludens/tap
brew install --cask bkt2gh
```

설치 확인:

```bash
bkt2gh --help
```

## 빠른 시작

1. 설정 파일 생성:

```bash
bkt2gh configure
```

1. 이전 계획 확인:

```bash
bkt2gh migrate-preview
```

1. 실제 마이그레이션 실행:

```bash
bkt2gh migrate
```

다른 Bitbucket workspace를 임시로 지정:

```bash
bkt2gh migrate-preview --workspace my-workspace
```

## 설정

`bkt2gh configure`는 OS 사용자 config 경로에 암호화된 `config.yaml`을 만듭니다. 암호화 키는 OS credential store/keychain에 저장합니다.

기본 config 경로:

- Linux: `$XDG_CONFIG_HOME/bkt2gh/config.yaml`, 또는 `~/.config/bkt2gh/config.yaml`
- macOS: `~/Library/Application Support/bkt2gh/config.yaml`
- Windows: `%AppData%\bkt2gh\config.yaml`

필수 값:

- Bitbucket username
- Bitbucket app password
- Bitbucket workspace
- GitHub token
- GitHub owner 또는 organization

설정 우선순위:

1. 환경변수
2. 암호화된 `config.yaml`

즉, 암호화된 `config.yaml`에 값이 있어도 같은 이름의 환경변수가 있으면 환경변수가 사용됩니다.

지원하는 환경변수:

```dotenv
BITBUCKET_USERNAME=you@example.com
BITBUCKET_APP_PASSWORD=your-bitbucket-app-password
BITBUCKET_WORKSPACE=your-workspace
GITHUB_TOKEN=your-github-token
GITHUB_OWNER=your-github-user-or-org
```

## 토큰 권한

Bitbucket app password 권한:

- Account: Read
- Workspace membership: Read
- Projects: Read
- Repositories: Read

GitHub token 권한:

- Metadata: Read-only
- Administration: Read and write
- Contents: Read and write

GitHub fine-grained token을 쓰는 경우 `GITHUB_OWNER`가 가리키는 사용자 또는 조직에 저장소를 만들 수 있어야 합니다.

## 사용법

```text
Usage:
  bkt2gh configure
  bkt2gh migrate-preview [--workspace name]
  bkt2gh migrate [--workspace name]

Commands:
  configure        create or update encrypted config.yaml interactively
  migrate-preview  preview migration plan without creating or pushing
  migrate          migrate selected Bitbucket repositories to GitHub

Flags:
  -h, --help show help
```

### `configure`

대화형 입력으로 암호화된 `config.yaml`을 생성하거나 갱신합니다.

```bash
bkt2gh configure
```

### `migrate-preview`

Bitbucket 저장소를 조회하고, 사용자가 선택한 저장소의 이전 계획을 출력합니다.

```bash
bkt2gh migrate-preview
```

옵션:

- `--workspace name`: 암호화된 config의 `BITBUCKET_WORKSPACE` 대신 사용할 workspace

### `migrate`

Bitbucket 저장소를 조회하고, 사용자가 선택한 저장소를 GitHub로 이전합니다.

```bash
bkt2gh migrate
```

옵션:

- `--workspace name`: 암호화된 config의 `BITBUCKET_WORKSPACE` 대신 사용할 workspace

## 저장소 선택

`migrate` 실행 시 저장소 선택 화면이 나옵니다.

명령:

- 숫자: 해당 저장소 선택/해제
- `1,3`: 여러 저장소 선택/해제
- `all`: 현재 보이는 저장소 전체 선택
- `none`: 현재 보이는 저장소 전체 해제
- `filter text`: 이름 또는 slug 기준 필터
- `done`: 선택 완료

## 공개 범위 정책

저장소 선택 후 GitHub 저장소 공개 범위 정책을 고릅니다.

- `all-private`: 모든 GitHub 저장소를 private으로 생성
- `all-public`: 모든 GitHub 저장소를 public으로 생성
- `follow-source`: Bitbucket 저장소의 공개/비공개 상태를 따름

## Preview

Preview는 Bitbucket/GitHub API를 호출해 계획을 확인하지만, 저장소를 만들거나 Git 명령을 실행하지 않습니다.

```bash
bkt2gh migrate-preview
```

확인하는 항목:

- Bitbucket 저장소 목록 조회 가능 여부
- GitHub token과 owner 접근 가능 여부
- 대상 GitHub 저장소 이름 사용 가능 여부
- 공개 범위 정책 적용 결과

## 실제 마이그레이션 동작

`migrate`를 실행하면 선택한 저장소마다 다음 순서로 처리합니다.

1. Bitbucket 저장소를 임시 디렉터리에 `git clone --mirror`로 복제
2. `git-lfs`가 있으면 `git lfs fetch --all` 시도
3. GitHub 저장소 생성
4. origin을 GitHub clone URL로 변경
5. `git-lfs`가 있으면 `git lfs push --all origin` 시도
6. `git push --mirror origin` 실행
7. 임시 디렉터리 정리

이미 같은 이름의 GitHub 저장소가 있으면 덮어쓰지 않고 건너뜁니다.

## 개발

테스트:

```bash
go test ./...
```

빌드:

```bash
go build -o bkt2gh ./cmd/bkt2gh
```

라이선스:

- 프로젝트: [MIT](LICENSE)
- 서드파티 고지: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
- 필요하면 서드파티 라이선스 원문 생성:
  `go run github.com/google/go-licenses@latest save ./... --save_path=third_party_licenses`

## 주의 사항

- 대상 GitHub 저장소가 이미 있으면 overwrite하지 않습니다.
- GitHub 저장소 이름은 Bitbucket 저장소 slug를 사용합니다.
- 암호화된 `config.yaml`은 저장소 밖 OS 사용자 config 경로에 저장됩니다.
- `git-lfs`가 없으면 LFS 처리는 건너뛰고 일반 Git mirror push만 진행합니다.
