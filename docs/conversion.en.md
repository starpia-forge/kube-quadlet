# Quadlet to Kubernetes Conversion Rules

This document describes the currently implemented conversion rules for `kube-quadlet`. The tool statically analyzes Quadlet files (`.container`, `.pod`, `.volume`) and generates Kubernetes manifests.

## General Rules

*   **Labels:** All generated objects include the label `app.kubernetes.io/name` set to the unit name (filename without extension).
*   **Replicas:** Deployments default to 1 replica.
*   **Service:** A Service is created if `PublishPort` is specified in a `.container` or `.pod` unit. The service type is `ClusterIP`.

## Container Unit (`.container`)

A `.container` unit is converted to a Kubernetes `Deployment`. If it exposes ports, a `Service` is also created.

### Basic Fields

| Quadlet Field | Kubernetes Mapping | Notes |
| :--- | :--- | :--- |
| `Image` | `spec.template.spec.containers[0].image` | The container image. |
| `Exec` | `spec.template.spec.containers[0].args` | Arguments to the entrypoint. Parsed as a shell command string. |
| `Entrypoint` | `spec.template.spec.containers[0].command` | Overrides the image entrypoint. If set, `Exec` becomes the arguments to this command. |
| `Environment` | `spec.template.spec.containers[0].env` | Key-value pairs for environment variables. |
| `WorkingDir` | `spec.template.spec.containers[0].workingDir` | The working directory inside the container. |

### Networking (`PublishPort`)

If `PublishPort` is present, a `Service` is created.

*   **Format:** `[hostPort:]containerPort`
*   **Mapping:**
    *   `containerPort` -> `Service.spec.ports[].targetPort` & `Container.ports[].containerPort`
    *   `hostPort` (or `containerPort` if omitted) -> `Service.spec.ports[].port`
*   **Protocol:** TCP (default).

### Storage (`Volume`)

Volumes are mapped to `volumeMounts` in the container and `volumes` in the Pod spec.

*   **Format:** `[source:]destination[:options]`
*   **Source Types:**
    *   **Empty:** Maps to `emptyDir`.
    *   **Absolute/Relative Path:** Maps to `hostPath`.
    *   **Name:** Maps to a `PersistentVolumeClaim` (PVC). If the source ends in `.volume`, the suffix is removed to find the PVC name.
*   **Options:**
    *   `ro`: Sets `readOnly: true` on the volume mount.

### Health Checks

The following fields map to the container's `livenessProbe`:

| Quadlet Field | Kubernetes `livenessProbe` Field |
| :--- | :--- |
| `HealthCmd` | `exec.command` | parsed as arguments |
| `HealthInterval` | `periodSeconds` | |
| `HealthTimeout` | `timeoutSeconds` | |
| `HealthStartPeriod` | `initialDelaySeconds` | |
| `HealthRetries` | `failureThreshold` | |

### Resources

| Quadlet Field | Kubernetes Mapping |
| :--- | :--- |
| `Memory` | `resources.limits.memory`, `resources.requests.memory` | Sets both limit and request. |

### Security Context

Fields map to `spec.template.spec.containers[0].securityContext`.

| Quadlet Field | Kubernetes Mapping |
| :--- | :--- |
| `User` | `runAsUser` | Must be numeric. |
| `Group` | `runAsGroup` | Must be numeric. |
| `ReadOnly` | `readOnlyRootFilesystem` | Boolean. |
| `NoNewPrivileges` | `allowPrivilegeEscalation` | Set to `false`. |
| `AddCapability` | `capabilities.add` | |
| `DropCapability` | `capabilities.drop` | |

### Pod Association (`Pod`)

*   If a `.container` file contains a `Pod` key referencing a `.pod` file in the same directory, it is intended to be aggregated into that Pod.
*   **Warning:** If you convert such a `.container` file directly (e.g., `kube-quadlet convert myapp.container`), it will be converted as a **standalone Deployment**, and the `Pod` field will be ignored with a warning.

## Pod Unit (`.pod`)

A `.pod` unit converts to a `Deployment` (representing the Pod) and optionally a `Service`. It aggregates all `.container` files in the same directory that reference it.

### Aggregation Logic
1.  Scans the directory for `.container` files.
2.  Parses each file to check the `Pod` field.
3.  If `Pod` matches the `.pod` filename (with or without extension), the container is added to the generated Deployment's Pod spec.

### Fields

| Quadlet Field | Kubernetes Mapping | Notes |
| :--- | :--- | :--- |
| `Volume` | `spec.template.spec.volumes` | Adds volumes to the Pod spec. **Note:** These are not automatically mounted into containers; containers must mount them explicitly using their own `Volume` field. |
| `PublishPort` | `Service.spec.ports` | Creates a Service exposing these ports. |

## Volume Unit (`.volume`)

A `.volume` unit converts to a `PersistentVolumeClaim` (PVC).

### Fields

| Quadlet Field | Kubernetes Mapping | Notes |
| :--- | :--- | :--- |
| `VolumeName` | `metadata.name` | Overrides the default name derived from the filename. |
| `Label` | `metadata.labels` | |

### Defaults
*   **Access Modes:** `ReadWriteOnce`
*   **Storage Request:** `1Gi`
