# Infrastructure Resilience Engine

> 인프라 복원력 테스트 애플리케이션을 구축하기 위한 완전히 불변인 프레임워크

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-latest-brightgreen.svg)](.kiro/specs/infrastructure-resilience-engine/)

한국어 | [English](README.md)

## 개요

Infrastructure Resilience Engine은 인프라 복원력 테스트 애플리케이션을 구축하기 위한 **완전히 불변인 프레임워크**입니다. 핵심 원칙은 **Core 수정 제로** - 모든 확장(새로운 환경, 플러그인, 저장소 백엔드, 모니터링 전략)은 Core 코드베이스를 건드리지 않고 잘 정의된 인터페이스를 통해 추가됩니다.

**현재 상태**: 🚧 개발 중 - 명세 단계 완료

- ✅ 요구사항 정의 완료 (20개 요구사항)
- ✅ 설계 완료 (98개 정확성 속성)
- ✅ 구현 계획 준비 완료 (6단계, 50+ 작업)
- 🚧 Phase 1: Core 인터페이스 및 모델 (진행 중)

### 주요 기능

- 🔒 **불변 Core**: Core를 수정하지 않고 새로운 환경, 플러그인, 기능 추가
- 🌍 **환경 독립적**: docker-compose, Kubernetes, ECS, Nomad 등을 위한 통합 추상화
- 🔌 **플러그인 아키텍처**: 라이프사이클 훅, 롤백 지원, 의존성을 갖춘 풍부한 플러그인 시스템
- 📊 **포괄적인 관찰 가능성**: 내장된 로깅, 메트릭, 분산 추적
- 🔐 **보안 우선**: 인증, 감사 로깅, 시크릿 관리
- 🧪 **테스트 친화적**: 광범위한 모킹 및 테스트 유틸리티 포함
- ⚡ **고성능**: 워커 풀, 배칭, 효율적인 이벤트 스트리밍

### 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                    Core (불변)                               │
│                                                              │
│  인터페이스: EnvironmentAdapter, Plugin, ExecutionEngine,    │
│              Monitor, Reporter, EventBus, Config 등          │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │ 구현
        ┌───────────────────┴───────────────────┐
        │                                       │
┌───────▼────────┐                    ┌────────▼────────┐
│   어댑터       │                    │    플러그인     │
│                │                    │                 │
│ - Compose      │                    │ - Kill          │
│ - Kubernetes   │                    │ - Restart       │
│ - ECS          │                    │ - Backup        │
│ - Nomad        │                    │ - Scale         │
└────────────────┘                    └─────────────────┘
        │                                       │
        └───────────────────┬───────────────────┘
                            │ 조합
                    ┌───────▼────────┐
                    │  애플리케이션  │
                    │                │
                    │ - GrimOps      │
                    │ - NecroOps     │
                    │ - BackupOps    │
                    └────────────────┘
```

## 빠른 시작

### 사전 요구사항

- Go 1.21 이상
- Docker (docker-compose 어댑터용)
- kubectl (Kubernetes 어댑터용, 선택사항)

### 설치

```bash
go get github.com/yourusername/infrastructure-resilience-engine
```

### 기본 사용법

#### 1. 간단한 플러그인 만들기

```go
package main

import (
    "context"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/interfaces"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

type HelloPlugin struct{}

func (p *HelloPlugin) Metadata() types.PluginMetadata {
    return types.PluginMetadata{
        Name:        "hello",
        Version:     "1.0.0",
        Description: "간단한 헬로 월드 플러그인",
    }
}

func (p *HelloPlugin) Execute(ctx interfaces.PluginContext, resource types.Resource) error {
    ctx.Logger.Info("플러그인에서 안녕하세요!", 
        types.Field{Key: "resource", Value: resource.Name})
    return nil
}

// 다른 Plugin 인터페이스 메서드 구현...
```

#### 2. 어댑터와 함께 사용하기

```go
package main

import (
    "context"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/adapters/compose"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/engine"
)

func main() {
    // 어댑터 생성
    adapter, err := compose.NewComposeAdapter(compose.AdapterConfig{
        ComposeFile: "docker-compose.yml",
    })
    if err != nil {
        panic(err)
    }
    
    // 실행 엔진 생성
    engine := engine.NewExecutionEngine()
    
    // 플러그인 등록
    plugin := &HelloPlugin{}
    engine.RegisterPlugin(plugin)
    
    // 리소스 목록 조회
    resources, err := adapter.ListResources(context.Background(), types.ResourceFilter{})
    if err != nil {
        panic(err)
    }
    
    // 첫 번째 리소스에 플러그인 실행
    if len(resources) > 0 {
        result, err := engine.Execute(context.Background(), types.ExecutionRequest{
            PluginName: "hello",
            Resource:   resources[0],
        })
        if err != nil {
            panic(err)
        }
        
        fmt.Printf("실행 결과: %+v\n", result)
    }
}
```

## 예제 애플리케이션

### GrimOps (카오스 엔지니어링)

GrimOps는 프레임워크 위에 구축된 카오스 엔지니어링 애플리케이션입니다:

```bash
# 컨테이너 종료
grimops attack redis --plugin kill

# 네트워크 지연 주입
grimops attack api --plugin network-delay --latency 500ms

# CPU 스트레스
grimops attack worker --plugin cpu-stress --cores 2 --duration 60s
```

### NecroOps (자가 치유)

NecroOps는 동일한 프레임워크 위에 구축된 자가 치유 애플리케이션입니다:

```bash
# 감시 및 자동 치유
necroops heal --watch --config necroops.yaml

# 수동 재시작
necroops heal redis --plugin restart

# 장애 시 스케일 업
necroops heal api --plugin scale --replicas 3
```

## 핵심 개념

### 리소스 모델

프레임워크는 환경 독립적인 리소스 모델을 사용합니다:

```go
type Resource struct {
    ID          string              // 고유 식별자
    Name        string              // 사람이 읽을 수 있는 이름
    Kind        string              // 리소스 타입 (container, pod, task 등)
    Labels      map[string]string   // 선택을 위한 레이블
    Annotations map[string]string   // 추가 메타데이터
    Status      ResourceStatus      // 현재 상태
    Spec        ResourceSpec        // 스펙
    Metadata    map[string]interface{} // 환경별 데이터
}
```

### 플러그인 라이프사이클

플러그인은 풍부한 라이프사이클을 따릅니다:

```
1. Validate    - 실행 가능 여부 확인
2. PreExecute  - 롤백을 위한 스냅샷 생성
3. Execute     - 실제 작업 수행
4. PostExecute - 후처리
5. Cleanup     - 실패 시에도 항상 실행
6. Rollback    - 선택사항, 실패 시 지원되면 실행
```

### 실행 전략

프레임워크는 플러그인 가능한 실행 전략을 지원합니다:

- **SimpleStrategy**: 직접 실행
- **RetryStrategy**: 지수 백오프로 재시도
- **CircuitBreakerStrategy**: 연쇄 장애 방지
- **RateLimitStrategy**: 실행 속도 제한

### 워크플로우

의존성이 있는 다단계 워크플로우 정의:

```yaml
workflow:
  name: chaos-and-heal
  steps:
    - name: kill-redis
      plugin: kill
      resource: redis
      
    - name: wait-for-failure
      plugin: wait
      depends_on: [kill-redis]
      
    - name: restart-redis
      plugin: restart
      resource: redis
      depends_on: [wait-for-failure]
      on_error: abort
```

## 개발

### 프로젝트 구조

```
infrastructure-resilience-engine/
├── cmd/
│   ├── grimops/          # 카오스 엔지니어링 앱
│   └── necroops/         # 자가 치유 앱
├── pkg/
│   ├── core/
│   │   ├── types/        # 데이터 모델
│   │   ├── interfaces/   # Core 인터페이스
│   │   ├── engine/       # 실행 엔진
│   │   ├── monitor/      # 모니터링 시스템
│   │   ├── reporter/     # 리포팅 시스템
│   │   ├── eventbus/     # 이벤트 버스
│   │   ├── config/       # 설정
│   │   ├── registry/     # 플러그인 레지스트리
│   │   └── testing/      # 테스트 유틸리티
│   ├── adapters/
│   │   ├── compose/      # docker-compose 어댑터
│   │   └── k8s/          # Kubernetes 어댑터
│   └── plugins/
│       ├── kill/         # Kill 플러그인
│       ├── restart/      # Restart 플러그인
│       ├── delay/        # 네트워크 지연 플러그인
│       └── healthmonitor/ # 헬스 모니터 플러그인
├── docs/
│   ├── plugin-development.md
│   ├── adapter-development.md
│   └── api-reference.md
├── examples/
│   ├── simple-plugin/
│   ├── custom-adapter/
│   └── workflow/
└── .kiro/specs/infrastructure-resilience-engine/
    ├── requirements.md
    ├── design.md
    └── tasks.md
```

### 소스에서 빌드하기

```bash
# 저장소 클론
git clone https://github.com/yourusername/infrastructure-resilience-engine.git
cd infrastructure-resilience-engine

# 의존성 설치
go mod download

# 빌드
make build

# 테스트 실행
make test

# race detector로 실행
make test-race

# 린트
make lint
```

### 테스트 실행

```bash
# 단위 테스트
go test ./...

# 통합 테스트
go test -tags=integration ./...

# 속성 기반 테스트
go test -tags=property ./...

# 커버리지
go test -cover ./...
```

## 문서

- [요구사항](.kiro/specs/infrastructure-resilience-engine/requirements.md)
- [설계](.kiro/specs/infrastructure-resilience-engine/design.md)
- [플러그인 개발 가이드](docs/plugin-development.md)
- [어댑터 개발 가이드](docs/adapter-development.md)
- [API 레퍼런스](docs/api-reference.md)

## 기여하기

기여를 환영합니다! 자세한 내용은 [CONTRIBUTING.md](CONTRIBUTING.md)를 참조하세요.

### 개발 워크플로우

1. 저장소 포크
2. 기능 브랜치 생성 (`git checkout -b feature/amazing-feature`)
3. 변경사항 작성
4. 테스트 실행 (`make test`)
5. 변경사항 커밋 (`git commit -m 'feat: add amazing feature'`)
6. 브랜치에 푸시 (`git push origin feature/amazing-feature`)
7. Pull Request 열기

### 코드 스타일

- [Effective Go](https://go.dev/doc/effective_go) 따르기
- `gofmt`와 `goimports` 사용
- 새 기능에 대한 테스트 작성
- 공개 API 문서화

## 라이선스

이 프로젝트는 MIT 라이선스에 따라 라이선스가 부여됩니다 - 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 감사의 말

- Chaos Mesh, Pumba, LitmusChaos에서 영감을 받음
- Kiroween 해커톤을 위해 제작
- DevOps 커뮤니티에 특별한 감사

## 연락처

- GitHub Issues: [https://github.com/yourusername/infrastructure-resilience-engine/issues](https://github.com/yourusername/infrastructure-resilience-engine/issues)
- Email: your.email@example.com

---

**인프라 복원력 테스트를 위해 ❤️로 제작**
