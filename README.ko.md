# bkt2gh

![bkt2gh](bkt-to-gh.jpg)

<p align="center"><strong>Bitbucket Cloud 저장소를 GitHub로 옮기는 Go CLI</strong></p>

<p align="center">
<a href="#주요-기능"><b>주요 기능</b></a> | <a href="#요구-사항"><b>요구 사항</b></a> | <a href="#설치"><b>설치</b></a> | <a href="#빠른-시작"><b>빠른 시작</b></a><br/>
<a href="#설정"><b>설정</b></a> | <a href="#사용법"><b>사용법</b></a> | <a href="#preview"><b>Preview</b></a> | <a href="#실제-마이그레이션-동작"><b>실제 마이그레이션 동작</b></a><br/>
<a href="#개발"><b>개발</b></a> | <a href="#라이선스"><b>라이선스</b></a> | <a href="README.md"><b>English</b></a>
</p>

## 주요 기능

- Bitbucket Cloud workspace의 저장소 목록 조회
- 터미널에서 이전할 저장소 선택
- GitHub 저장소 생성
- mirror clone/push 기반 Git 이력 이전
- GitHub 저장소 공개 범위 정책 선택
- migration preview 사전 점검
- OS 사용자 config 경로의 암호화된 설정과 환경변수 override

## 요구 사항

- Git
- Bitbucket Cloud 계정과 app password
- GitHub token

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

2. 이전 계획 확인:

```bash
bkt2gh migrate-preview
```

3. 실제 마이그레이션 실행:

```bash
bkt2gh migrate
```

다른 Bitbucket workspace를 임시로 지정:

```bash
bkt2gh migrate-preview --workspace my-workspace
```

### 에이전트 스킬

bkt2gh는 코딩 에이전트에 설정 및 마이그레이션 실행 방법을 알려주는 스킬(`skills/bkt2gh/SKILL.md`)을 포함합니다. 다음 명령으로 설치하세요:

```bash
npx skills add ludens/bkt-to-gh
```

이 명령은 감지된 에이전트(Claude Code, Codex, Cursor, Pi 등)에 `bkt2gh` 스킬을 추가합니다. `-g`(전역)나 `--skill bkt2gh` 같은 옵션은 [skills](https://github.com/vercel-labs/skills)를 참고하세요.

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

### 토큰 권한

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
  --workspace name  Bitbucket workspace (이번 실행에서 config 값 override)
  -h, --help        show help
```

### `configure`

암호화된 `config.yaml`을 대화형으로 생성하거나 갱신합니다. [설정](#설정) 참고.

### `migrate-preview`

Bitbucket 저장소를 조회하고, 저장소를 만들거나 push하지 않고 이전 계획을 출력합니다. [Preview](#preview) 참고.

### `migrate`

Bitbucket 저장소를 조회하고 선택한 저장소를 GitHub로 이전합니다. [실제 마이그레이션 동작](#실제-마이그레이션-동작) 참고.

### 저장소 선택

`migrate` 또는 `migrate-preview` 실행 시 저장소 선택 화면이 나옵니다.

명령:

- 숫자: 해당 저장소 선택/해제
- `1,3`: 여러 저장소 선택/해제
- `all`: 현재 보이는 저장소 전체 선택
- `none`: 현재 보이는 저장소 전체 해제
- `filter text`: 이름 또는 slug 기준 필터
- `done`: 선택 완료

### 공개 범위 정책

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
2. GitHub 저장소 생성
3. origin을 GitHub clone URL로 변경
4. `git push --mirror origin` 실행
5. 임시 디렉터리 정리

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

## 라이선스

- 프로젝트: [MIT](LICENSE)
- 서드파티 고지: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
