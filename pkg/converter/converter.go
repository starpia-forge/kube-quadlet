package converter

import (
	"fmt"
	"math"
	"kuadlet/pkg/quadlet"
	"os"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ConvertVolume(v *quadlet.VolumeUnit, name string) ([]runtime.Object, error) {
	pvcName := name
	if v.Volume.VolumeName != "" {
		pvcName = v.Volume.VolumeName
	}

	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   pvcName,
			Labels: v.Volume.Label,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	return []runtime.Object{pvc}, nil
}

// ConvertContainer now accepts a volume registry to lookup actual VolumeName
func ConvertContainer(c *quadlet.ContainerUnit, name string, volumeRegistry map[string]*quadlet.VolumeUnit) ([]runtime.Object, error) {
	container, volumes, servicePorts, err := createContainerSpec(c, name, volumeRegistry)
	if err != nil {
		return nil, err
	}

	labels := map[string]string{
		"app.kubernetes.io/name": name,
	}
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{*container},
					Volumes:    volumes,
				},
			},
		},
	}

	var objects []runtime.Object
	objects = append(objects, deployment)

	if len(servicePorts) > 0 {
		service := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Ports:    servicePorts,
				Type:     corev1.ServiceTypeClusterIP,
			},
		}
		objects = append(objects, service)
	}

	return objects, nil
}

func ConvertPod(p *quadlet.PodUnit, containers []*quadlet.ContainerUnit, containerNames []string, name string, volumeRegistry map[string]*quadlet.VolumeUnit) ([]runtime.Object, error) {
	var objects []runtime.Object
	labels := map[string]string{
		"app.kubernetes.io/name": name,
	}

	var podContainers []corev1.Container
	var podVolumes []corev1.Volume
	var podVolumeMounts []corev1.VolumeMount

	for i, volSpec := range p.Pod.Volume {
		vol, mount, err := parseVolumeSpec(volSpec, fmt.Sprintf("pod-vol-%d", i), volumeRegistry)
		if err != nil {
			return nil, err
		}
		podVolumes = append(podVolumes, *vol)
		podVolumeMounts = append(podVolumeMounts, *mount)
	}

	for i, c := range containers {
		cName := containerNames[i]
		container, cVolumes, _, err := createContainerSpec(c, cName, volumeRegistry)
		if err != nil {
			return nil, err
		}

		for j := range cVolumes {
			oldName := cVolumes[j].Name
			newName := fmt.Sprintf("%s-%s", cName, oldName)
			cVolumes[j].Name = newName

			for k := range container.VolumeMounts {
				if container.VolumeMounts[k].Name == oldName {
					container.VolumeMounts[k].Name = newName
				}
			}
		}

		// Mount pod-level volumes into the container
		container.VolumeMounts = append(container.VolumeMounts, podVolumeMounts...)

		podContainers = append(podContainers, *container)
		podVolumes = append(podVolumes, cVolumes...)
	}

	replicas := int32(1)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: podContainers,
					Volumes:    podVolumes,
				},
			},
		},
	}
	objects = append(objects, deployment)

	var servicePorts []corev1.ServicePort
	seenServicePorts := make(map[int32]string)

	for i, portSpec := range p.Pod.PublishPort {
		_, _, sPort, err := parsePortSpec(portSpec, fmt.Sprintf("pod-port-%d", i))
		if err != nil {
			continue
		}

		// Check for duplicate host port
		if definedIn, ok := seenServicePorts[sPort.Port]; ok {
			return nil, fmt.Errorf("duplicate port definition detected in Pod: port %d is already defined in %s", sPort.Port, definedIn)
		}
		seenServicePorts[sPort.Port] = fmt.Sprintf("PublishPort index %d", i)

		servicePorts = append(servicePorts, *sPort)
	}

	if len(servicePorts) > 0 {
		service := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Ports:    servicePorts,
				Type:     corev1.ServiceTypeClusterIP,
			},
		}
		objects = append(objects, service)
	}

	return objects, nil
}

func ConvertKube(k *quadlet.KubeUnit, name string) ([]runtime.Object, error) {
	safeName := sanitize(name)
	// #nosec G705
	fmt.Fprintf(os.Stderr, "Warning: .kube unit %s detected. This unit type wraps a Kubernetes YAML file. Kuadlet cannot fully convert this wrapper to a Kubernetes manifest as it IS already a Kubernetes manifest wrapper. Please apply the referenced YAML file directly: %s\n", safeName, k.Kube.Yaml)
	return nil, nil
}

func ConvertNetwork(n *quadlet.NetworkUnit, name string) ([]runtime.Object, error) {
	safeName := sanitize(name)
	// #nosec G705
	fmt.Fprintf(os.Stderr, "Warning: .network unit %s detected. Kubernetes handles networking differently (CNI). Podman network configurations do not directly map to Kubernetes resources.\n", safeName)
	return nil, nil
}

func ConvertImage(i *quadlet.ImageUnit, name string) ([]runtime.Object, error) {
	safeName := sanitize(name)
	// #nosec G705
	fmt.Fprintf(os.Stderr, "Warning: .image unit %s detected. Kubernetes pulls images automatically on Pod scheduling. Explicit image pull units are not typically needed or supported as standalone resources.\n", safeName)
	return nil, nil
}

func ConvertBuild(b *quadlet.BuildUnit, name string) ([]runtime.Object, error) {
	safeName := sanitize(name)
	// #nosec G705
	fmt.Fprintf(os.Stderr, "Warning: .build unit %s detected. Kubernetes does not support building images at runtime. Please build and push the image to a registry before deploying.\n", safeName)
	return nil, nil
}

func ConvertArtifact(a *quadlet.ArtifactUnit, name string) ([]runtime.Object, error) {
	safeName := sanitize(name)
	// #nosec G705
	fmt.Fprintf(os.Stderr, "Warning: .artifact unit %s detected. This is an experimental Quadlet feature not directly supported in Kubernetes.\n", safeName)
	return nil, nil
}

func createContainerSpec(c *quadlet.ContainerUnit, name string, volumeRegistry map[string]*quadlet.VolumeUnit) (*corev1.Container, []corev1.Volume, []corev1.ServicePort, error) {
	var env []corev1.EnvVar
	for k, v := range c.Container.Environment {
		env = append(env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	var command []string
	var args []string
	if c.Container.Exec != "" {
		parsedArgs, err := SplitArgs(c.Container.Exec)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse Exec: %w", err)
		}
		args = parsedArgs
	}

	if c.Container.Entrypoint != "" {
		command = []string{c.Container.Entrypoint}
	}

	var containerPorts []corev1.ContainerPort
	var servicePorts []corev1.ServicePort

	// Deduplication check for Service Ports
	// Key: port (host port)
	seenServicePorts := make(map[int32]string)

	for i, portSpec := range c.Container.PublishPort {
		cPort, _, sPort, err := parsePortSpec(portSpec, fmt.Sprintf("port-%d", i))
		if err != nil {
			continue
		}

		// Check for duplicate host port
		if definedIn, ok := seenServicePorts[sPort.Port]; ok {
			return nil, nil, nil, fmt.Errorf("duplicate port definition detected: port %d is already defined in %s", sPort.Port, definedIn)
		}
		seenServicePorts[sPort.Port] = fmt.Sprintf("PublishPort index %d", i)

		containerPorts = append(containerPorts, *cPort)
		servicePorts = append(servicePorts, *sPort)
	}

	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume

	for i, volSpec := range c.Container.Volume {
		vol, mount, err := parseVolumeSpec(volSpec, fmt.Sprintf("vol-%d", i), volumeRegistry)
		if err != nil {
			continue
		}
		volumes = append(volumes, *vol)
		volumeMounts = append(volumeMounts, *mount)
	}

	// Probes
	var livenessProbe *corev1.Probe
	if c.Container.HealthCmd != "" {
		// "none" disables it
		if c.Container.HealthCmd != "none" {
			var probeCmd []string
			// If it looks like a JSON array or explicitly starts with sh -c, maybe we could trust it,
			// but to be safe and consistent with typical shell usage in HealthCmd, we wrap it.
			// However, if it IS an array (starts with [), we should probably parse it as such?
			// Quadlet docs say HealthCmd is "command to run". Podman treats it as CMD-SHELL if string.
			// So wrapping in sh -c is the correct default for a string.
			// Exception: if it starts with `["`, it might be an exec array?
			// But Quadlet parser reads it as a raw string.
			// Let's assume it's a shell command string.

			// Optimization: if it already starts with "sh -c" or "/bin/sh -c", we might split it naively?
			// No, safer to just wrap it unless we want to parse it.
			// But wait, if the user wrote `HealthCmd=curl ...`, we want `sh -c "curl ..."`.
			// If we use SplitArgs, we get `["curl", "..."]`. This FAILS for `curl ... || exit 1`.
			// So we MUST use `sh -c`.
			probeCmd = []string{"sh", "-c", c.Container.HealthCmd}

			livenessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: probeCmd,
					},
				},
			}

				if c.Container.HealthInterval != "" {
					if d, err := time.ParseDuration(c.Container.HealthInterval); err == nil {
						if d.Seconds() > math.MaxInt32 {
							livenessProbe.PeriodSeconds = math.MaxInt32
						} else {
							livenessProbe.PeriodSeconds = int32(d.Seconds())
						}
					}
				}
				if c.Container.HealthTimeout != "" {
					if d, err := time.ParseDuration(c.Container.HealthTimeout); err == nil {
						if d.Seconds() > math.MaxInt32 {
							livenessProbe.TimeoutSeconds = math.MaxInt32
						} else {
							livenessProbe.TimeoutSeconds = int32(d.Seconds())
						}
					}
				}
				if c.Container.HealthStartPeriod != "" {
					if d, err := time.ParseDuration(c.Container.HealthStartPeriod); err == nil {
						if d.Seconds() > math.MaxInt32 {
							livenessProbe.InitialDelaySeconds = math.MaxInt32
						} else {
							livenessProbe.InitialDelaySeconds = int32(d.Seconds())
						}
					}
				}
				if c.Container.HealthRetries > 0 {
					if c.Container.HealthRetries > math.MaxInt32 {
						livenessProbe.FailureThreshold = math.MaxInt32
					} else {
						livenessProbe.FailureThreshold = int32(c.Container.HealthRetries)
					}
				}
		}
	}

	// Resources
	resources := corev1.ResourceRequirements{}
	if c.Container.Memory != "" {
		if q, err := resource.ParseQuantity(c.Container.Memory); err == nil {
			resources.Limits = corev1.ResourceList{corev1.ResourceMemory: q}
			resources.Requests = corev1.ResourceList{corev1.ResourceMemory: q}
		}
	}

	// SecurityContext
	securityContext := &corev1.SecurityContext{}
	hasSecurityContext := false

	if c.Container.User != "" {
		if uid, err := strconv.ParseInt(c.Container.User, 10, 64); err == nil {
			securityContext.RunAsUser = &uid
			hasSecurityContext = true
		}
	}
	if c.Container.Group != "" {
		if gid, err := strconv.ParseInt(c.Container.Group, 10, 64); err == nil {
			securityContext.RunAsGroup = &gid
			hasSecurityContext = true
		}
	}
	if c.Container.ReadOnly {
		ro := true
		securityContext.ReadOnlyRootFilesystem = &ro
		hasSecurityContext = true
	}
	if c.Container.NoNewPrivileges {
		nnp := false
		securityContext.AllowPrivilegeEscalation = &nnp
		hasSecurityContext = true
	}

	if len(c.Container.AddCapability) > 0 || len(c.Container.DropCapability) > 0 {
		caps := &corev1.Capabilities{}
		for _, cap := range c.Container.AddCapability {
			caps.Add = append(caps.Add, corev1.Capability(strings.ToUpper(cap)))
		}
		for _, cap := range c.Container.DropCapability {
			caps.Drop = append(caps.Drop, corev1.Capability(strings.ToUpper(cap)))
		}
		securityContext.Capabilities = caps
		hasSecurityContext = true
	}

	var sc *corev1.SecurityContext
	if hasSecurityContext {
		sc = securityContext
	}

	container := &corev1.Container{
		Name:            name,
		Image:           c.Container.Image,
		Command:         command,
		Args:            args,
		Env:             env,
		Ports:           containerPorts,
		WorkingDir:      c.Container.WorkingDir,
		VolumeMounts:    volumeMounts,
		LivenessProbe:   livenessProbe,
		Resources:       resources,
		SecurityContext: sc,
	}

	return container, volumes, servicePorts, nil
}

func parsePortSpec(spec string, name string) (*corev1.ContainerPort, int, *corev1.ServicePort, error) {
	parts := strings.Split(spec, ":")
	var containerPortStr, hostPortStr string

	if len(parts) == 1 {
		containerPortStr = parts[0]
		hostPortStr = parts[0]
	} else if len(parts) == 2 {
		hostPortStr = parts[0]
		containerPortStr = parts[1]
		if hostPortStr == "" {
			hostPortStr = containerPortStr
		}
	} else {
		containerPortStr = parts[len(parts)-1]
		hostPortStr = parts[len(parts)-2]
	}

	cPort, err := strconv.Atoi(containerPortStr)
	if err != nil {
		return nil, 0, nil, err
	}

	hPort, err := strconv.Atoi(hostPortStr)
	if err != nil {
		hPort = cPort
	}

	if cPort > 65535 || cPort < 0 {
		return nil, 0, nil, fmt.Errorf("container port %d out of valid range (0-65535)", cPort)
	}
	if hPort > 65535 || hPort < 0 {
		return nil, 0, nil, fmt.Errorf("host port %d out of valid range (0-65535)", hPort)
	}

	cp := &corev1.ContainerPort{
		Name:          name,
		ContainerPort: int32(cPort), // #nosec G109
		Protocol:      corev1.ProtocolTCP,
	}

	sp := &corev1.ServicePort{
		Name:       name,
		Port:       int32(hPort), // #nosec G109
		TargetPort: intstr.FromInt(cPort),
		Protocol:   corev1.ProtocolTCP,
	}

	return cp, hPort, sp, nil
}

func parseVolumeSpec(spec string, name string, volumeRegistry map[string]*quadlet.VolumeUnit) (*corev1.Volume, *corev1.VolumeMount, error) {
	parts := strings.Split(spec, ":")
	var source, dest string
	var readOnly bool

	if len(parts) == 1 {
		dest = parts[0]
		source = ""
	} else {
		last := parts[len(parts)-1]
		isOption := false
		if strings.Contains(last, ",") || last == "ro" || last == "rw" || last == "z" || last == "Z" {
			isOption = true
			if last == "ro" || strings.Contains(last, "ro,") || strings.Contains(last, ",ro") {
				readOnly = true
			}
		}

		if isOption {
			dest = parts[len(parts)-2]
			if len(parts) > 2 {
				source = strings.Join(parts[:len(parts)-2], ":")
			}
		} else {
			dest = parts[len(parts)-1]
			source = strings.Join(parts[:len(parts)-1], ":")
		}
	}

	vm := &corev1.VolumeMount{
		Name:      name,
		MountPath: dest,
		ReadOnly:  readOnly,
	}

	var vol *corev1.Volume
	if source == "" {
		vol = &corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
	} else if strings.HasPrefix(source, "/") || strings.HasPrefix(source, ".") {
		vol = &corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: source,
				},
			},
		}
	} else {
		claimName := source
		if strings.HasSuffix(source, ".volume") {
			// Cross-referencing logic
			volumeName := strings.TrimSuffix(source, ".volume")

			// Lookup in registry
			if volumeRegistry != nil {
				if v, ok := volumeRegistry[volumeName]; ok {
					if v.Volume.VolumeName != "" {
						claimName = v.Volume.VolumeName
					} else {
						claimName = volumeName
					}
				} else {
					// Fallback to filename if not found in registry (e.g. implicitly defined or external)
					// Log warning?
					// fmt.Fprintf(os.Stderr, "Warning: Referenced volume %s not found in input files. Using filename as claimName.\n", source)
					claimName = volumeName
				}
			} else {
				claimName = volumeName
			}
		}
		vol = &corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
			},
		}
	}

	return vol, vm, nil
}

func sanitize(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "\r", "")
}
