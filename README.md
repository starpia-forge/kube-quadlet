# kube-quadlet

`kube-quadlet` is a lightweight CLI tool designed to bridge the gap between Podman's systemd integration and Kubernetes. It statically parses Podman Quadlet configuration files (such as `.container`, `.pod`, and `.volume`) and automatically generates equivalent Kubernetes YAML manifests.

Unlike the native `podman kube generate` command, which requires the containers to be actively running or created in the local Podman database, `kube-quadlet` performs a **pure static translation**. You only need the text files to generate your Kubernetes Deployments, Services, and PersistentVolumeClaims.

## 🎯 Project Goals

* **Static Conversion:** Generate Kubernetes YAML directly from Quadlet text files without needing a running container daemon.
* **Seamless Migration:** Provide a straightforward path for developers moving workloads from single-node Rootless Podman environments to full Kubernetes clusters.
* **CI/CD Friendly:** Enable easy integration into automated pipelines where running a Podman service is not feasible.

## ✨ Key Features

* **`.container` Parsing:** Translates standard Quadlet container configurations into Kubernetes `Deployment` and `Pod` specifications.
* **Network & Port Mapping:** Automatically extracts `PublishPort` and other network directives to generate corresponding Kubernetes `Service` manifests.
* **Volume Translation:** Converts Quadlet volume mounts into Kubernetes `PersistentVolumeClaim` (PVC) and `Volume` mounts.
* **Environment Variables:** Maps `Environment` and `EnvironmentFile` keys directly to Kubernetes `env` and `envFrom` specs.

## 🚀 Quick Start (Example)

```bash
# Convert a single Quadlet container file to a Kubernetes manifest
kube-quadlet convert ./my-app.container > deployment.yaml

# Convert a Quadlet pod file
kube-quadlet convert ./my-stack.pod > my-stack.yaml
```

## 📚 Documentation

For detailed information on how Quadlet fields are mapped to Kubernetes objects, please refer to the conversion rules:

*   [Conversion Rules (English)](docs/conversion.en.md)
*   [Conversion Rules (Korean)](docs/conversion.ko.md)

## 🤝 Conventions

* **Commit Messages:** All commit messages must adhere to the [Conventional Commits](https://www.conventionalcommits.org/) specification (e.g., `feat: add container parser`, `fix: correct port mapping`, `docs: update readme`).

## 📜 License

All code was written by `jules`.
