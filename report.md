# Kuadlet to Kubernetes Conversion Test Report

This report documents the testing of the `kuadlet` tool, which converts Podman Quadlet files to Kubernetes Manifests. Tests are performed on a local kind cluster.

## Supported and Tested Features

| Section | Key | Status | Notes |
|---|---|---|---|
| `[Container]` | `Image` | Works | Successfully translated to `image:` field in PodSpec. |
| `[Container]` | `PublishPort` | Works | Translated to both `containerPort` in Deployment and K8s `Service`. Also works with `IP:HostPort:ContainerPort` formats. |
| `[Container]` | `Environment` | Works | Converted to `env:` array correctly. Tricky quotes are preserved. Blank continuations omit the value. |
| `[Container]` | `User` & `Group` | Works | Converted to `runAsUser` and `runAsGroup` in `securityContext`. |
| `[Container]` | `Memory` | Works | Translated to `resources.limits.memory` and `requests.memory`. |
| `[Container]` | `AddCapability` / `DropCapability` | Works | Translated to `securityContext.capabilities.add` / `drop`. |
| `[Container]` | `ReadOnly` | Works | Translated to `readOnlyRootFilesystem: true`. |
| `[Container]` | `NoNewPrivileges` | Works | Translated to `allowPrivilegeEscalation: false`. |
| `[Container]` | `Health*` (Probe) | Partial | Translated to `livenessProbe.exec`. Note: Command strings are naive-split and not wrapped in a shell. E.g., `||` and `exit` are passed as literal arguments to the command, which may fail in k8s. |
| `[Volume]` | `VolumeName` | Buggy | Translated to `PersistentVolumeClaim.metadata.name`. However, `[Container]`'s `Volume=` reference uses the filename without `.volume` suffix, causing a mismatch if `VolumeName` is different from the filename! |
| `[Container]` | `Volume` (Named) | Buggy | Uses the Quadlet base filename (e.g. `test2-vol` from `test2-vol.volume:/data:rw`) as the PVC claimName, ignoring the actual `VolumeName` specified inside the `.volume` file. |
| `[Container]` | `Volume` (HostPath) | Works | Properly parses host path bounds (e.g., `/host/path:/container/path:ro`) to K8s HostPath. |
| `[Container]` | `Volume` (Multiple) | Works | Can mount multiple volumes within a single container successfully. |
| `[Container]` | `Exec` | Works | Translated to K8s `args:`. Note that Podman `Exec` is a single string split by space naively. |
| `[Container]` | `Pod` | Partial | Successfully assigns the container to the Pod's Deployment. However, if the `.container` file is passed to `kuadlet convert` along with the `.pod` file, it *also* generates a standalone duplicate Deployment for the container (with a warning). |
| `[Pod]` | `PodName` | Buggy | Ignored! The generated Deployment is named after the filename (e.g., `test4-pod`), not the `PodName` key. |
| `[Pod]` | `PublishPort` | Works | Successfully translated to `Service` matching the Pod deployment. |
| `[Pod]` | `Volume` | Buggy | Has the same filename vs `VolumeName` bug as `[Container]`. Also, it adds the Volume to the Pod spec, but completely fails to mount it into the member containers (`volumeMounts` is missing). |
| `[Kube]` / `[Network]` | N/A | Works | Emits correct warnings indicating that K8s CNI or wrapper approaches cannot be directly converted. No YAML is outputted. |
| `[Container]` | Unknown Keys | Works | Keys not explicitly supported (like `FakeKey`) emit standard `Warning: Unknown key in [Container]: FakeKey` to `stderr` and are ignored. |

## Complex Service Testing (Real-world Scenario)

We simulated a complex service containing `app.container`, `db.container`, `test.volume`, and `test.network` residing in a single directory. The goal was to convert this directory and deploy it directly to K8s.

### Findings from Complex Scenario

1.  **Directory Conversion & Output Structure:** `kuadlet convert <dir>` successfully discovers and processes all recognized `.container`, `.volume`, `.pod`, etc., files within the directory. It outputs a combined multi-document YAML stream by default.
2.  **Network Units are Skipped:** As expected, the `.network` file was skipped with a warning. For K8s, standard CNI handles networking, and DNS resolves services.
3.  **Service Discovery (DNS Resolution):** Podman relies on `.network` for container-to-container DNS. In K8s, this is handled by `Service` objects. Since `kuadlet` generates a K8s `Service` for any container with a `PublishPort`, container communication (e.g., App to DB via hostname `db`) works automatically in K8s *if* both expose a port and have matching filenames.
4.  **Volume Binding Failure (Critical Issue):**
    *   In the complex scenario, the DB container was stuck in a `Pending` state.
    *   **Reason:** The `.volume` file had `VolumeName=test-db-pvc`. The generated PVC was named `test-db-pvc`. However, `db.container` referenced it as `Volume=test.volume:/var/lib/...`.
    *   `kuadlet` incorrectly parsed the `db.container` reference, looking for a PVC named `test` (based on the filename `test.volume` stripped of `.volume`) instead of the actual `VolumeName` (`test-db-pvc`).
    *   This confirms that **volumes using `VolumeName` that differs from the file's base name are currently broken and unusable without manual YAML modification.**
5.  **Conflicting Ports Edge Case:**
    *   If a `.container` defines multiple `PublishPort` entries that point to the same K8s container port (e.g., `8080:80` and `9090:80`), `kuadlet` generates multiple entries for `containerPort: 80` in the PodSpec. K8s warns about duplicate port definitions but still deploys the pod.
    *   However, if there are duplicate ports mapped identically in the `PublishPort` configuration, the resulting K8s `Service` becomes invalid (`Duplicate value`), and K8s outright rejects the Service creation. `kuadlet` currently lacks deduplication logic.
