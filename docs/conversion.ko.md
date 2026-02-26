# Quadlet to Kubernetes 변환 규칙

이 문서는 `kube-quadlet`의 현재 구현된 변환 규칙을 설명합니다. 이 도구는 Quadlet 파일(`.container`, `.pod`, `.volume`)을 정적으로 분석하여 Kubernetes 매니페스트를 생성합니다.

## 일반 규칙

*   **Labels:** 생성된 모든 객체에는 유닛 이름(확장자 제외)으로 설정된 `app.kubernetes.io/name` 라벨이 포함됩니다.
*   **Replicas:** Deployment의 기본 복제본(replicas) 수는 1입니다.
*   **Service:** `.container` 또는 `.pod` 유닛에 `PublishPort`가 지정된 경우 Service가 생성됩니다. 서비스 타입은 `ClusterIP`입니다.

## 컨테이너 유닛 (`.container`)

`.container` 유닛은 Kubernetes `Deployment`로 변환됩니다. 포트를 노출하는 경우 `Service`도 함께 생성됩니다.

### 기본 필드 (Basic Fields)

| Quadlet Field | Kubernetes Mapping | 비고 |
| :--- | :--- | :--- |
| `Image` | `spec.template.spec.containers[0].image` | 컨테이너 이미지. |
| `Exec` | `spec.template.spec.containers[0].args` | 엔트리포인트에 대한 인자(arguments). 쉘 커맨드 문자열로 파싱됩니다. |
| `Entrypoint` | `spec.template.spec.containers[0].command` | 이미지 엔트리포인트를 덮어씁니다. 설정된 경우, `Exec`은 이 커맨드의 인자가 됩니다. |
| `Environment` | `spec.template.spec.containers[0].env` | 환경 변수 키-값 쌍. |
| `WorkingDir` | `spec.template.spec.containers[0].workingDir` | 컨테이너 내부의 작업 디렉토리. |

### 네트워킹 (`PublishPort`)

`PublishPort`가 존재하면 `Service`가 생성됩니다.

*   **형식:** `[hostPort:]containerPort`
*   **매핑:**
    *   `containerPort` -> `Service.spec.ports[].targetPort` 및 `Container.ports[].containerPort`
    *   `hostPort` (생략 시 `containerPort`) -> `Service.spec.ports[].port`
*   **프로토콜:** TCP (기본값).

### 스토리지 (`Volume`)

볼륨은 컨테이너의 `volumeMounts`와 Pod spec의 `volumes`로 매핑됩니다.

*   **형식:** `[source:]destination[:options]`
*   **소스 유형:**
    *   **Empty:** `emptyDir`로 매핑됩니다.
    *   **절대/상대 경로:** `hostPath`로 매핑됩니다.
    *   **이름:** `PersistentVolumeClaim` (PVC)으로 매핑됩니다. 소스가 `.volume`으로 끝나는 경우 접미사를 제거하여 PVC 이름을 찾습니다.
*   **옵션:**
    *   `ro`: 볼륨 마운트에 `readOnly: true`를 설정합니다.

### 헬스 체크 (Health Checks)

다음 필드들은 컨테이너의 `livenessProbe`로 매핑됩니다:

| Quadlet Field | Kubernetes `livenessProbe` Field |
| :--- | :--- |
| `HealthCmd` | `exec.command` | 인자로 파싱됨 |
| `HealthInterval` | `periodSeconds` | |
| `HealthTimeout` | `timeoutSeconds` | |
| `HealthStartPeriod` | `initialDelaySeconds` | |
| `HealthRetries` | `failureThreshold` | |

### 리소스 (Resources)

| Quadlet Field | Kubernetes Mapping |
| :--- | :--- |
| `Memory` | `resources.limits.memory`, `resources.requests.memory` | limit과 request를 모두 설정합니다. |

### 보안 컨텍스트 (Security Context)

필드들은 `spec.template.spec.containers[0].securityContext`로 매핑됩니다.

| Quadlet Field | Kubernetes Mapping | 비고 |
| :--- | :--- | :--- |
| `User` | `runAsUser` | 숫자여야 합니다. |
| `Group` | `runAsGroup` | 숫자여야 합니다. |
| `ReadOnly` | `readOnlyRootFilesystem` | Boolean. |
| `NoNewPrivileges` | `allowPrivilegeEscalation` | `false`로 설정됩니다. |
| `AddCapability` | `capabilities.add` | |
| `DropCapability` | `capabilities.drop` | |

### Pod 연결 (`Pod`)

*   `.container` 파일에 같은 디렉토리의 `.pod` 파일을 참조하는 `Pod` 키가 포함된 경우, 해당 Pod로 집계되도록 의도된 것입니다.
*   **경고:** 이러한 `.container` 파일을 직접 변환(예: `kube-quadlet convert myapp.container`)하면 **독립적인 Deployment**로 변환되며, `Pod` 필드는 경고와 함께 무시됩니다.

## Pod 유닛 (`.pod`)

`.pod` 유닛은 `Deployment`(Pod를 나타냄)와 선택적으로 `Service`로 변환됩니다. 이 유닛은 자신을 참조하는 같은 디렉토리의 모든 `.container` 파일을 집계합니다.

### 집계 로직
1.  디렉토리에서 `.container` 파일을 스캔합니다.
2.  각 파일을 파싱하여 `Pod` 필드를 확인합니다.
3.  `Pod` 필드가 `.pod` 파일명(확장자 포함 또는 미포함)과 일치하면, 해당 컨테이너는 생성된 Deployment의 Pod spec에 추가됩니다.

### 필드

| Quadlet Field | Kubernetes Mapping | 비고 |
| :--- | :--- | :--- |
| `Volume` | `spec.template.spec.volumes` | Pod spec에 볼륨을 추가합니다. **참고:** 이 볼륨들은 컨테이너에 자동으로 마운트되지 않으며, 컨테이너가 자신의 `Volume` 필드를 사용하여 명시적으로 마운트해야 합니다. |
| `PublishPort` | `Service.spec.ports` | 이 포트들을 노출하는 Service를 생성합니다. |

## 볼륨 유닛 (`.volume`)

`.volume` 유닛은 `PersistentVolumeClaim` (PVC)으로 변환됩니다.

### 필드

| Quadlet Field | Kubernetes Mapping | 비고 |
| :--- | :--- | :--- |
| `VolumeName` | `metadata.name` | 파일명에서 파생된 기본 이름을 덮어씁니다. |
| `Label` | `metadata.labels` | |

### 기본값
*   **Access Modes:** `ReadWriteOnce`
*   **Storage Request:** `1Gi`
