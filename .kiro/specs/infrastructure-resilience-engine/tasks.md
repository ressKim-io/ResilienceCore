# 구현 계획

이 문서는 Infrastructure Resilience Engine의 구현 작업 목록입니다. 각 작업은 Core를 절대 수정하지 않고 완전히 불변인 뼈대를 만드는 것을 목표로 합니다.

## Phase 1: Core 인터페이스 및 모델 (1주차)

**목표**: 모든 Core 인터페이스와 데이터 모델 정의

- [x] 1. 프로젝트 초기 설정
  - Go 모듈 초기화 (`go mod init`)
  - 디렉토리 구조 생성 (pkg/core, pkg/adapters, pkg/plugins, cmd)
  - 기본 Makefile 작성 (build, test, lint 타겟)
  - GitHub Actions CI 설정 (테스트, 린트)
  - _요구사항: 모든 요구사항의 기반_

- [x] 2. 데이터 모델 정의
  - Resource 구조체 및 관련 타입 정의 (ResourceStatus, ResourceSpec, Condition 등)
  - HealthCheckConfig, Port, Volume, VolumeSource 정의
  - Metrics 구조체 정의 (CPUMetrics, MemoryMetrics, NetworkMetrics, DiskMetrics)
  - Event 구조체 정의
  - _요구사항: 2.1, 2.3, 2.4_

- [x] 2.1 데이터 모델 속성 기반 테스트 작성
  - **Property 6: Resource contains environment-agnostic fields**
  - **Property 8: Resource supports arbitrary metadata**
  - **검증: 요구사항 2.1, 2.3**

- [x] 3. EnvironmentAdapter 인터페이스 정의
  - EnvironmentAdapter 인터페이스 정의
  - AdapterConfig, ResourceFilter, LabelSelector 정의
  - ResourceEvent, DeleteOptions, ExecOptions, ExecResult 정의
  - AdapterInfo 정의
  - _요구사항: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 3.1 EnvironmentAdapter 속성 기반 테스트 작성
  - **Property 17: Adapter provides all required methods**
  - **Property 21: Adapter interface is sufficient for all platforms**
  - **검증: 요구사항 4.1, 4.5**

- [x] 4. Plugin 인터페이스 정의
  - Plugin 인터페이스 정의 (Metadata, Initialize, Validate, PreExecute, Execute, PostExecute, Cleanup, Rollback)
  - PluginMetadata, PluginConfig, PluginContext 정의
  - Snapshot 인터페이스 정의
  - ExecutionResult, ExecutionStatus 정의
  - ProgressUpdate 정의
  - _요구사항: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_

- [x] 4.1 Plugin 속성 기반 테스트 작성
  - **Property 11: Plugin metadata is complete**
  - **Property 12: Plugin lifecycle hooks execute in order**
  - **Property 13: Rollback is invoked on failure**
  - **검증: 요구사항 3.1, 3.2, 3.3**

- [x] 5. ExecutionEngine 인터페이스 정의
  - ExecutionEngine 인터페이스 정의
  - ExecutionRequest, ExecutionStrategy 인터페이스 정의
  - Future 인터페이스 정의
  - Workflow, WorkflowStep, WorkflowResult 정의
  - ErrorHandler 인터페이스, ErrorAction 정의
  - 내장 전략 정의 (SimpleStrategy, RetryStrategy, CircuitBreakerStrategy, RateLimitStrategy)
  - _요구사항: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_


- [x] 5.1 ExecutionEngine 속성 기반 테스트 작성
  - **Property 22: Execution strategy is applied**
  - **Property 23: Concurrency limits are respected**
  - **Property 24: Timeout cancels execution**
  - **검증: 요구사항 5.1, 5.2, 5.3**

- [x] 6. Monitor 인터페이스 정의
  - Monitor 인터페이스 정의
  - HealthCheckStrategy 인터페이스 정의
  - HealthStatus, HealthStatusType, HealthCheck 정의
  - 내장 전략 정의 (HTTPHealthCheckStrategy, TCPHealthCheckStrategy, ExecHealthCheckStrategy)
  - _요구사항: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 6.1 Monitor 속성 기반 테스트 작성
  - **Property 28: Health check uses registered strategy**
  - **Property 32: Custom health check strategies are supported**
  - **검증: 요구사항 6.1, 6.5**

- [x] 7. Reporter 인터페이스 정의
  - Reporter 인터페이스 정의
  - ExecutionRecord, EventQuery, ExecutionQuery 정의
  - Statistics, PluginStatistics, ResourceStatistics 정의
  - StorageBackend 인터페이스, Query, SortField 정의
  - ReportFormatter 인터페이스 정의
  - 내장 포맷터 정의 (JSONFormatter, MarkdownFormatter, HTMLFormatter)
  - _요구사항: 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 7.1 Reporter 속성 기반 테스트 작성
  - **Property 33: Execution record is complete**
  - **Property 35: Statistics are calculated correctly**
  - **Property 37: Custom storage backends are supported**
  - **검증: 요구사항 7.1, 7.3, 7.5**

- [ ] 8. EventBus 인터페이스 정의
  - EventBus 인터페이스 정의
  - EventFilter, Subscription 인터페이스 정의
  - _요구사항: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 8.1 EventBus 속성 기반 테스트 작성
  - **Property 38: Events are delivered to all matching subscribers**
  - **Property 39: Subscription receives only matching events**
  - **Property 40: Unsubscribe stops event delivery**
  - **검증: 요구사항 8.1, 8.2, 8.3**

- [ ] 9. Config 인터페이스 정의
  - Config 인터페이스 정의
  - ConfigProvider 인터페이스, ConfigChange 정의
  - 내장 프로바이더 정의 (YAMLConfigProvider, EnvironmentConfigProvider, ConsulConfigProvider, VaultConfigProvider)
  - _요구사항: 9.1, 9.2, 9.3, 9.4, 9.5_

- [ ] 9.1 Config 속성 기반 테스트 작성
  - **Property 43: Multiple config providers are supported**
  - **Property 44: Config merge follows precedence**
  - **Property 47: Custom config providers are supported**
  - **검증: 요구사항 9.1, 9.2, 9.5**


- [ ] 10. 보안 인터페이스 정의
  - AuthorizationProvider 인터페이스, Action 정의
  - AuditLogger 인터페이스, AuditEntry, ActionResult 정의
  - SecretProvider 인터페이스 정의
  - _요구사항: 10.1, 10.2, 10.3, 10.4, 10.5_

- [ ] 10.1 보안 속성 기반 테스트 작성
  - **Property 48: Authorization is checked before execution**
  - **Property 50: Secrets are not exposed in logs**
  - **Property 52: Custom security providers are supported**
  - **검증: 요구사항 10.1, 10.3, 10.5**

- [ ] 11. Observability 인터페이스 정의
  - ObservabilityProvider 인터페이스 정의
  - Logger 인터페이스, Field 정의
  - MetricsCollector 인터페이스, Counter, Gauge, Histogram 인터페이스 정의
  - Tracer 인터페이스, Span 인터페이스, SpanOption, SpanConfig 정의
  - _요구사항: 11.1, 11.2, 11.3, 11.4, 11.5_

- [ ] 11.1 Observability 속성 기반 테스트 작성
  - **Property 53: Logs contain structured fields**
  - **Property 55: Trace span is created and propagated**
  - **Property 57: Custom observability providers are supported**
  - **검증: 요구사항 11.1, 11.3, 11.5**

- [ ] 12. PluginRegistry 인터페이스 정의
  - PluginRegistry 인터페이스 정의
  - PluginLoader 인터페이스, PluginValidator 인터페이스 정의
  - PluginFilter 정의
  - _요구사항: 13.1, 13.2, 13.3, 13.4, 13.5_

- [ ] 12.1 PluginRegistry 속성 기반 테스트 작성
  - **Property 59: Plugins are discovered from directories**
  - **Property 60: Plugin validation occurs on registration**
  - **Property 63: Hot-reload updates plugins without restart**
  - **검증: 요구사항 13.1, 13.2, 13.5**

- [ ] 13. 테스트 유틸리티 패키지 작성
  - MockEnvironmentAdapter 구현
  - MockPlugin 구현
  - MockMonitor, MockReporter, MockEventBus 구현
  - TestHarness 구현
  - ResourceBuilder 구현
  - WorkflowTestRunner 구현
  - _요구사항: 12.1, 12.2, 12.3, 12.4, 12.5_

- [ ] 13.1 테스트 유틸리티 속성 기반 테스트 작성
  - **Property 58: Mock implementations exist for all interfaces**
  - **검증: 요구사항 12.5**

- [ ] 14. Phase 1 체크포인트
  - 모든 테스트가 통과하는지 확인
  - 사용자에게 질문이 있으면 문의


## Phase 2: 기본 구현체 (2주차)

**목표**: Core 컴포넌트의 기본 구현 완성

- [ ] 15. DefaultExecutionEngine 구현
  - 플러그인 등록/해제 구현
  - Execute 메서드 구현 (라이프사이클 훅 순서 보장)
  - ExecuteAsync 메서드 구현 (Future 반환)
  - 동시성 제어 구현 (세마포어, 리소스 잠금)
  - 타임아웃 처리 구현
  - 에러 처리 및 롤백 구현
  - _요구사항: 5.1, 5.2, 5.3, 14.1, 14.2, 14.3, 15.5_

- [ ] 15.1 실행 전략 구현
  - SimpleStrategy 구현
  - RetryStrategy 구현 (지수 백오프)
  - CircuitBreakerStrategy 구현
  - RateLimitStrategy 구현
  - _요구사항: 5.1, 14.1_

- [ ] 15.2 ExecutionEngine 단위 테스트 작성
  - 플러그인 등록/해제 테스트
  - 라이프사이클 훅 순서 테스트
  - 동시성 제한 테스트
  - 타임아웃 테스트
  - 롤백 테스트

- [ ] 15.3 ExecutionEngine 속성 기반 테스트 작성
  - **Property 12: Plugin lifecycle hooks execute in order**
  - **Property 13: Rollback is invoked on failure**
  - **Property 23: Concurrency limits are respected**
  - **Property 24: Timeout cancels execution**
  - **Property 64: Retryable errors trigger retry with backoff**
  - **Property 73: Resource locking prevents concurrent execution**
  - **검증: 요구사항 3.2, 3.3, 5.2, 5.3, 14.1, 15.5**

- [ ] 16. Workflow 실행 구현
  - ExecuteWorkflow 메서드 구현
  - 의존성 그래프 구축 및 순환 의존성 감지
  - 위상 정렬 구현
  - 순차/병렬 실행 구현
  - 에러 핸들러 구현 (Continue, Abort, Retry, Rollback)
  - 조건부 실행 구현
  - _요구사항: 5.4, 5.5, 16.1, 16.2, 16.3, 16.4, 16.5_

- [ ] 16.1 Workflow 속성 기반 테스트 작성
  - **Property 25: Workflow respects dependencies**
  - **Property 74: Workflow supports sequential and parallel execution**
  - **Property 75: Dependencies are waited for**
  - **Property 76: Step error handler is executed**
  - **Property 78: Conditional execution works correctly**
  - **검증: 요구사항 5.4, 5.5, 16.1, 16.2, 16.3, 16.5**

- [ ] 17. DefaultMonitor 구현
  - CheckHealth 메서드 구현 (전략 선택 및 실행)
  - WaitForCondition 메서드 구현 (폴링)
  - WaitForHealthy 메서드 구현
  - CollectMetrics 메서드 구현 (어댑터 위임)
  - WatchEvents 메서드 구현
  - 헬스체크 전략 등록/조회 구현
  - _요구사항: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ] 17.1 내장 헬스체크 전략 구현
  - HTTPHealthCheckStrategy 구현
  - TCPHealthCheckStrategy 구현
  - ExecHealthCheckStrategy 구현
  - _요구사항: 6.5_

- [ ] 17.2 Monitor 단위 테스트 작성
  - 헬스체크 전략 선택 테스트
  - 조건 대기 테스트
  - 메트릭 수집 테스트

- [ ] 17.3 Monitor 속성 기반 테스트 작성
  - **Property 28: Health check uses registered strategy**
  - **Property 29: Condition waiting polls until met or timeout**
  - **Property 30: Metrics collection delegates to adapter**
  - **검증: 요구사항 6.1, 6.2, 6.3**


- [ ] 18. DefaultReporter 구현
  - RecordEvent, RecordExecution 메서드 구현
  - QueryEvents, QueryExecutions 메서드 구현
  - ComputeStatistics 메서드 구현 (MTTR, 백분위수 계산)
  - GenerateReport 메서드 구현 (포맷터 선택)
  - 저장소 설정 및 포맷터 등록 구현
  - _요구사항: 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ] 18.1 InMemoryStorage 구현
  - Save, Load, Query, Delete 메서드 구현
  - 인메모리 맵 기반 저장소
  - _요구사항: 7.5_

- [ ] 18.2 내장 리포트 포맷터 구현
  - JSONFormatter 구현
  - MarkdownFormatter 구현
  - HTMLFormatter 구현 (선택사항)
  - _요구사항: 7.4_

- [ ] 18.3 Reporter 단위 테스트 작성
  - 이벤트/실행 기록 테스트
  - 쿼리 테스트
  - 통계 계산 테스트
  - 리포트 생성 테스트

- [ ] 18.4 Reporter 속성 기반 테스트 작성
  - **Property 33: Execution record is complete**
  - **Property 34: Event record is complete**
  - **Property 35: Statistics are calculated correctly**
  - **Property 36: Report uses registered formatter**
  - **검증: 요구사항 7.1, 7.2, 7.3, 7.4**

- [ ] 19. InMemoryEventBus 구현
  - Publish, PublishAsync 메서드 구현
  - Subscribe 메서드 구현 (필터링)
  - Unsubscribe 구현
  - Close 구현 (모든 채널 닫기)
  - _요구사항: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 19.1 EventBus 단위 테스트 작성
  - 이벤트 발행/구독 테스트
  - 필터링 테스트
  - 구독 취소 테스트
  - 종료 테스트

- [ ] 19.2 EventBus 속성 기반 테스트 작성
  - **Property 38: Events are delivered to all matching subscribers**
  - **Property 39: Subscription receives only matching events**
  - **Property 40: Unsubscribe stops event delivery**
  - **Property 41: Shutdown closes all channels gracefully**
  - **Property 42: Event filtering supports type, source, and metadata**
  - **검증: 요구사항 8.1, 8.2, 8.3, 8.4, 8.5**

- [ ] 20. DefaultConfig 구현
  - Get, GetString, GetInt, GetBool, GetDuration 메서드 구현
  - Set 메서드 구현
  - Watch 메서드 구현
  - AddProvider 메서드 구현 (우선순위 기반 병합)
  - _요구사항: 9.1, 9.2, 9.3, 9.4, 9.5_

- [ ] 20.1 YAMLConfigProvider 구현
  - YAML 파일 로딩
  - 파일 감시 (fsnotify)
  - _요구사항: 9.1_

- [ ] 20.2 EnvironmentConfigProvider 구현
  - 환경 변수 로딩
  - 접두사 지원
  - _요구사항: 9.1_

- [ ] 20.3 Config 단위 테스트 작성
  - 값 조회 테스트
  - 타입 변환 테스트
  - 병합 우선순위 테스트
  - 감시 테스트

- [ ] 20.4 Config 속성 기반 테스트 작성
  - **Property 43: Multiple config providers are supported**
  - **Property 44: Config merge follows precedence**
  - **Property 45: Config values are type-converted**
  - **Property 46: Config changes notify watchers**
  - **검증: 요구사항 9.1, 9.2, 9.3, 9.4**


- [ ] 21. DefaultPluginRegistry 구현
  - Register, Unregister 메서드 구현
  - Get, List 메서드 구현
  - Discover, DiscoverFromDirectory 메서드 구현
  - Validate, ValidateDependencies 메서드 구현 (순환 의존성 감지)
  - Reload 메서드 구현
  - _요구사항: 13.1, 13.2, 13.3, 13.4, 13.5, 18.1, 18.2, 18.3, 18.4, 18.5_

- [ ] 21.1 PluginRegistry 단위 테스트 작성
  - 플러그인 등록/해제 테스트
  - 검증 테스트
  - 의존성 검증 테스트
  - 순환 의존성 감지 테스트
  - 필터링 테스트

- [ ] 21.2 PluginRegistry 속성 기반 테스트 작성
  - **Property 60: Plugin validation occurs on registration**
  - **Property 61: Plugin retrieval returns plugin or error**
  - **Property 62: Plugin listing supports filtering**
  - **Property 84: Dependencies are declared in metadata**
  - **Property 85: Dependencies are verified on load**
  - **Property 86: Missing dependencies prevent registration**
  - **Property 87: Circular dependencies are detected**
  - **Property 88: Version constraints are validated**
  - **검증: 요구사항 13.2, 13.3, 13.4, 18.1, 18.2, 18.3, 18.4, 18.5**

- [ ] 22. 기본 Observability 구현
  - StdoutLogger 구현 (구조화된 로깅)
  - NoOpMetricsCollector 구현 (메트릭 무시)
  - NoOpTracer 구현 (트레이싱 무시)
  - _요구사항: 11.1, 11.2, 11.3, 11.4, 11.5_

- [ ] 22.1 Observability 속성 기반 테스트 작성
  - **Property 53: Logs contain structured fields**
  - **검증: 요구사항 11.1**

- [ ] 23. Phase 2 체크포인트
  - 모든 테스트가 통과하는지 확인
  - 사용자에게 질문이 있으면 문의

## Phase 3: 어댑터 구현 (3주차)

**목표**: docker-compose 및 Kubernetes 어댑터 구현

- [ ] 24. ComposeAdapter 구현
  - Initialize 메서드 구현 (Docker 클라이언트 초기화)
  - ListResources 메서드 구현 (docker-compose.yml 파싱, 컨테이너 목록)
  - GetResource 메서드 구현
  - StartResource, StopResource, RestartResource 메서드 구현
  - DeleteResource, CreateResource, UpdateResource 메서드 구현
  - WatchResources 메서드 구현 (Docker events API)
  - ExecInResource 메서드 구현
  - GetMetrics 메서드 구현 (docker stats API)
  - _요구사항: 4.1, 4.2, 4.3, 4.4_

- [ ] 24.1 docker-compose.yml 파서 구현
  - YAML 파싱
  - 서비스 정의를 Resource로 변환
  - Docker label 기반 매핑 (`com.docker.compose.service`)
  - _요구사항: 2.2_

- [ ] 24.2 ComposeAdapter 통합 테스트 작성
  - testcontainers-go 사용
  - 실제 컨테이너 생성/조작 테스트
  - 이벤트 감시 테스트

- [ ] 24.3 ComposeAdapter 속성 기반 테스트 작성
  - **Property 7: Adapter conversion produces valid Resources**
  - **Property 9: Resource status is normalized across environments**
  - **Property 18: Watch emits ResourceEvent objects**
  - **Property 19: Exec returns structured result**
  - **Property 20: Metrics are normalized**
  - **검증: 요구사항 2.2, 2.4, 4.2, 4.3, 4.4**


- [ ] 25. K8sAdapter 구현
  - Initialize 메서드 구현 (kubeconfig 로딩, clientset 초기화)
  - ListResources 메서드 구현 (Pod, Deployment 목록)
  - GetResource 메서드 구현
  - StartResource, StopResource, RestartResource 메서드 구현
  - DeleteResource, CreateResource, UpdateResource 메서드 구현
  - WatchResources 메서드 구현 (Informer 패턴)
  - ExecInResource 메서드 구현 (remotecommand)
  - GetMetrics 메서드 구현 (metrics-server API)
  - _요구사항: 4.1, 4.2, 4.3, 4.4_

- [ ] 25.1 Kubernetes 리소스 변환 구현
  - Pod를 Resource로 변환
  - Deployment를 Resource로 변환
  - Label selector 지원
  - _요구사항: 2.2_

- [ ] 25.2 K8sAdapter 통합 테스트 작성
  - kind (Kubernetes in Docker) 사용
  - 실제 Pod 생성/조작 테스트
  - Informer 테스트

- [ ] 25.3 K8sAdapter 속성 기반 테스트 작성
  - **Property 7: Adapter conversion produces valid Resources**
  - **Property 9: Resource status is normalized across environments**
  - **Property 18: Watch emits ResourceEvent objects**
  - **Property 19: Exec returns structured result**
  - **Property 20: Metrics are normalized**
  - **검증: 요구사항 2.2, 2.4, 4.2, 4.3, 4.4**

- [ ] 26. 어댑터 불변성 검증
  - ComposeAdapter와 K8sAdapter가 Core 수정 없이 작동하는지 확인
  - 새 어댑터 추가 시나리오 테스트 (예: MockCloudAdapter)
  - _요구사항: 1.1, 4.5_

- [ ] 26.1 어댑터 불변성 속성 기반 테스트 작성
  - **Property 1: Core remains unchanged when adding new environments**
  - **Property 21: Adapter interface is sufficient for all platforms**
  - **검증: 요구사항 1.1, 4.5**

- [ ] 27. Phase 3 체크포인트
  - 모든 테스트가 통과하는지 확인
  - 사용자에게 질문이 있으면 문의

## Phase 4: 예제 플러그인 구현 (4주차)

**목표**: 플러그인 시스템 검증을 위한 예제 플러그인 구현

- [ ] 28. KillPlugin 구현 (Chaos)
  - Metadata 구현
  - Initialize 구현 (설정 로딩)
  - Validate 구현 (리소스 존재 확인)
  - PreExecute 구현 (상태 스냅샷)
  - Execute 구현 (Compose: ContainerKill, K8s: Pod Delete)
  - PostExecute 구현
  - Cleanup 구현
  - Rollback 구현 (재시작)
  - _요구사항: 3.1, 3.2, 3.3, 3.6_

- [ ] 28.1 KillPlugin 설정 구조체 정의
  - Signal (SIGKILL, SIGTERM)
  - Force (bool)
  - WaitForDeletion (bool)
  - _요구사항: 3.1_

- [ ] 28.2 KillPlugin 단위 테스트 작성
  - Mock 어댑터 사용
  - 라이프사이클 훅 테스트
  - 환경별 실행 테스트

- [ ] 28.3 KillPlugin 통합 테스트 작성
  - 실제 컨테이너/Pod kill 테스트
  - 롤백 테스트

- [ ] 28.4 KillPlugin 속성 기반 테스트 작성
  - **Property 11: Plugin metadata is complete**
  - **Property 12: Plugin lifecycle hooks execute in order**
  - **Property 13: Rollback is invoked on failure**
  - **검증: 요구사항 3.1, 3.2, 3.3**


- [ ] 29. RestartPlugin 구현 (Healing)
  - Metadata 구현
  - Initialize 구현
  - Validate 구현
  - PreExecute 구현
  - Execute 구현 (Compose: ContainerRestart, K8s: Pod Delete + Wait)
  - PostExecute 구현 (헬스체크 대기)
  - Cleanup 구현
  - _요구사항: 3.1, 3.2, 3.6_

- [ ] 29.1 RestartPlugin 설정 구조체 정의
  - WaitHealthy (bool)
  - HealthTimeout (duration)
  - RestartTimeout (duration)
  - _요구사항: 3.1_

- [ ] 29.2 RestartPlugin 단위 테스트 작성
  - Mock 어댑터 사용
  - 헬스체크 대기 테스트

- [ ] 29.3 RestartPlugin 통합 테스트 작성
  - 실제 재시작 테스트
  - 헬스체크 확인 테스트

- [ ] 30. DelayPlugin 구현 (Chaos)
  - Metadata 구현
  - Initialize 구현
  - Validate 구현 (exec 권한 확인)
  - PreExecute 구현
  - Execute 구현 (tc 명령으로 네트워크 지연 주입)
  - Cleanup 구현 (tc 설정 제거)
  - _요구사항: 3.1, 3.2, 3.6_

- [ ] 30.1 DelayPlugin 설정 구조체 정의
  - Latency (duration)
  - Jitter (duration)
  - PacketLoss (percentage)
  - Duration (duration)
  - _요구사항: 3.1_

- [ ] 30.2 DelayPlugin 통합 테스트 작성
  - 네트워크 지연 측정 테스트
  - Cleanup 확인 테스트

- [ ] 31. HealthMonitorPlugin 구현 (Healing)
  - Metadata 구현
  - Initialize 구현
  - Execute 구현 (주기적 헬스체크, 실패 시 복구 트리거)
  - _요구사항: 3.1, 3.6_

- [ ] 31.1 HealthMonitorPlugin 설정 구조체 정의
  - Interval (duration)
  - UnhealthyThreshold (int)
  - RecoveryAction (string - restart, scale 등)
  - _요구사항: 3.1_

- [ ] 31.2 HealthMonitorPlugin 통합 테스트 작성
  - 자동 복구 사이클 테스트
  - 이벤트 발행 테스트

- [ ] 32. 플러그인 불변성 검증
  - 모든 플러그인이 Core 수정 없이 작동하는지 확인
  - 새 플러그인 추가 시나리오 테스트
  - _요구사항: 1.2_

- [ ] 32.1 플러그인 불변성 속성 기반 테스트 작성
  - **Property 2: Core remains unchanged when adding new plugins**
  - **검증: 요구사항 1.2**

- [ ] 33. Phase 4 체크포인트
  - 모든 테스트가 통과하는지 확인
  - 사용자에게 질문이 있으면 문의


## Phase 5: 애플리케이션 구축 (5주차)

**목표**: Core를 사용하여 GrimOps와 NecroOps 애플리케이션 구축

- [ ] 34. GrimOps CLI 구현
  - cobra를 사용한 CLI 프레임워크 구축
  - `attack` 명령 구현 (플러그인 실행)
  - `report` 명령 구현 (리포트 생성)
  - `list` 명령 구현 (타겟 목록)
  - 설정 파일 로딩 (grimops.yaml)
  - _요구사항: 모든 요구사항의 통합_

- [ ] 34.1 GrimOps 설정 파일 정의
  - 어댑터 설정 (compose 또는 k8s)
  - 플러그인 설정
  - 실행 전략 설정
  - _요구사항: 9.1, 9.2_

- [ ] 34.2 GrimOps main.go 작성
  - Core 컴포넌트 초기화
  - 어댑터 등록
  - 플러그인 등록 (Kill, Delay, Stress)
  - CLI 실행
  - _요구사항: 1.1, 1.2_

- [ ] 34.3 GrimOps E2E 테스트 작성
  - docker-compose 환경에서 전체 시나리오 테스트
  - kill → 상태 확인 → 리포트 생성

- [ ] 35. NecroOps CLI 구현
  - cobra를 사용한 CLI 프레임워크 구축
  - `heal` 명령 구현 (플러그인 실행)
  - `watch` 모드 구현 (백그라운드 감시)
  - `report` 명령 구현 (리포트 생성)
  - 설정 파일 로딩 (necroops.yaml)
  - _요구사항: 모든 요구사항의 통합_

- [ ] 35.1 NecroOps 설정 파일 정의
  - 어댑터 설정
  - 플러그인 설정
  - 감시 설정 (interval, threshold)
  - _요구사항: 9.1, 9.2_

- [ ] 35.2 NecroOps main.go 작성
  - Core 컴포넌트 초기화
  - 어댑터 등록
  - 플러그인 등록 (Restart, Scale, HealthMonitor)
  - EventBus 구독 (장애 감지)
  - CLI 실행
  - _요구사항: 1.1, 1.2, 8.1, 8.2_

- [ ] 35.3 NecroOps E2E 테스트 작성
  - docker-compose 환경에서 전체 시나리오 테스트
  - 장애 감지 → 자동 복구 → MTTR 측정

- [ ] 36. 통합 시나리오 구현
  - GrimOps + NecroOps 통합 시나리오
  - GrimOps가 redis kill → NecroOps가 자동 감지 및 재시작
  - MTTR 측정 및 리포트 생성
  - _요구사항: 모든 요구사항의 통합_

- [ ] 36.1 통합 시나리오 E2E 테스트 작성
  - docker-compose 환경 테스트
  - K8s 환경 테스트 (kind 사용)

- [ ] 37. 데모 환경 구성
  - docker-compose.yml 작성 (redis, api, nginx)
  - 샘플 API 작성 (Go, redis 의존)
  - 헬스체크 엔드포인트 구현
  - _요구사항: 데모용_

- [ ] 38. 데모 시나리오 스크립트 작성
  - 시나리오 1: Kill & Restart (5분)
  - 시나리오 2: Network Delay (선택사항)
  - 시나리오 3: K8s 환경 (선택사항)
  - _요구사항: 데모용_

- [ ] 39. Phase 5 체크포인트
  - 모든 테스트가 통과하는지 확인
  - 사용자에게 질문이 있으면 문의


## Phase 6: 문서화 및 마무리 (6주차)

**목표**: 문서 완성 및 프로젝트 마무리

- [ ] 40. README 업데이트
  - Quick Start 섹션 업데이트
  - 예제 코드 추가
  - 빌드 및 테스트 명령 추가
  - 기여 가이드 추가
  - _요구사항: 문서화_

- [ ] 41. 플러그인 개발 가이드 작성
  - 플러그인 인터페이스 설명
  - 플러그인 개발 단계별 가이드
  - 예제 플러그인 코드
  - 테스트 작성 가이드
  - _요구사항: 문서화_

- [ ] 42. 어댑터 개발 가이드 작성
  - EnvironmentAdapter 인터페이스 설명
  - 어댑터 개발 단계별 가이드
  - 예제 어댑터 코드
  - 테스트 작성 가이드
  - _요구사항: 문서화_

- [ ] 43. API 레퍼런스 문서 작성
  - godoc 주석 완성
  - 모든 공개 인터페이스 문서화
  - 예제 코드 추가
  - _요구사항: 문서화_

- [ ] 44. 예제 프로젝트 작성
  - examples/simple-plugin/ - 간단한 플러그인 예제
  - examples/custom-adapter/ - 커스텀 어댑터 예제
  - examples/workflow/ - 워크플로우 예제
  - _요구사항: 문서화_

- [ ] 45. 성능 테스트 및 최적화
  - 벤치마크 테스트 작성
  - 플러그인 실행 오버헤드 측정 (목표: <10ms)
  - 이벤트 전달 지연 측정 (목표: <1ms)
  - 동시 실행 성능 테스트 (목표: 1000+ concurrent)
  - 메모리 사용량 측정 (목표: <100MB idle)
  - _요구사항: 20.1, 20.2, 20.3, 20.4_

- [ ] 45.1 성능 속성 기반 테스트 작성
  - **Property 94: Pagination prevents memory exhaustion**
  - **Property 95: Worker pool limits resource consumption**
  - **Property 96: Writes are batched**
  - **검증: 요구사항 20.1, 20.2, 20.3**

- [ ] 46. 코드 품질 검증
  - golangci-lint 실행 및 모든 이슈 수정
  - go vet 실행
  - gofmt, goimports 실행
  - 테스트 커버리지 확인 (목표: 80%)
  - race detector 실행 (`go test -race`)
  - _요구사항: 품질 기준_

- [ ] 47. 최종 통합 테스트
  - 모든 단위 테스트 실행
  - 모든 통합 테스트 실행
  - 모든 속성 기반 테스트 실행 (100+ iterations)
  - 모든 E2E 테스트 실행
  - _요구사항: 모든 요구사항_

- [ ] 48. 릴리스 준비
  - CHANGELOG.md 작성
  - 버전 태그 생성 (v1.0.0)
  - GitHub Release 생성
  - 바이너리 빌드 및 업로드
  - _요구사항: 릴리스_

- [ ] 49. 최종 검증
  - 모든 98개 correctness properties가 통과하는지 확인
  - GrimOps와 NecroOps가 Core 수정 없이 작동하는지 확인
  - 새 환경 추가 시나리오 테스트 (MockCloudAdapter)
  - 새 플러그인 추가 시나리오 테스트
  - _요구사항: 1.1, 1.2, 모든 correctness properties_

- [ ] 50. Phase 6 완료
  - 모든 작업 완료 확인
  - 프로젝트 완성!

## 선택적 작업 (Phase 7+)

이 작업들은 MVP 이후 추가 개선을 위한 것입니다:

- [ ] 51. Web UI 구현
  - 대시보드 (실행 현황, 통계)
  - 플러그인 관리
  - 워크플로우 편집기
  - _요구사항: 향후 개선_

- [ ] 52. 추가 어댑터 구현
  - ECSAdapter (AWS ECS)
  - NomadAdapter (HashiCorp Nomad)
  - AzureContainerInstancesAdapter
  - _요구사항: 향후 개선_

- [ ] 53. 추가 플러그인 구현
  - CPUStressPlugin
  - MemoryStressPlugin
  - DiskFillPlugin
  - BackupPlugin
  - ScalePlugin
  - _요구사항: 향후 개선_

- [ ] 54. 고급 Observability 구현
  - PrometheusMetricsCollector
  - JaegerTracer
  - OpenTelemetry 통합
  - _요구사항: 향후 개선_

- [ ] 55. 분산 실행 지원
  - 분산 락 서비스 통합 (etcd, Consul)
  - 다중 인스턴스 조정
  - _요구사항: 향후 개선_

## 성공 기준

### 기능 요구사항
- [ ] 모든 20개 요구사항 구현 완료
- [ ] 모든 98개 correctness properties 통과
- [ ] GrimOps와 NecroOps가 Core 수정 없이 작동
- [ ] 새 환경 추가 시 Core 수정 불필요
- [ ] 새 플러그인 추가 시 Core 수정 불필요

### 품질 요구사항
- [ ] 단위 테스트 커버리지 ≥ 80%
- [ ] 모든 속성 기반 테스트 통과 (100+ iterations)
- [ ] docker-compose 및 K8s 통합 테스트 통과
- [ ] E2E 테스트 통과
- [ ] race detector 통과 (`go test -race`)
- [ ] golangci-lint 통과

### 문서 요구사항
- [ ] 모든 공개 인터페이스에 godoc 주석
- [ ] README with quick start
- [ ] 플러그인 개발 가이드
- [ ] 어댑터 개발 가이드
- [ ] API 레퍼런스
- [ ] 최소 3개 예제 프로젝트

### 성능 요구사항
- [ ] 플러그인 실행 오버헤드 < 10ms
- [ ] 이벤트 전달 지연 < 1ms
- [ ] 1000+ 동시 플러그인 실행 지원
- [ ] 10,000+ 리소스 per 어댑터 지원
- [ ] 유휴 메모리 사용량 < 100MB

## 참고사항

- **모든 작업 필수**: 테스트를 포함한 모든 작업이 필수입니다. 포괄적인 테스트 커버리지를 통해 Core의 불변성과 정확성을 보장합니다.
- **체크포인트**: 각 Phase 끝에 체크포인트가 있어 진행 상황을 확인하고 문제를 조기에 발견할 수 있습니다.
- **Core 불변성**: 모든 작업은 Core를 수정하지 않고 완료되어야 합니다. 이것이 프로젝트의 핵심 원칙입니다.
- **속성 기반 테스트**: 각 correctness property는 gopter를 사용한 속성 기반 테스트로 검증됩니다 (100+ iterations).
