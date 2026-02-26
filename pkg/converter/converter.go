package converter

import (
	"fmt"
	"math"
	"kuadlet/pkg/quadlet"
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

func ConvertContainer(c *quadlet.ContainerUnit, name string) ([]runtime.Object, error) {
	container, volumes, servicePorts, err := createContainerSpec(c, name)
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

func ConvertPod(p *quadlet.PodUnit, containers []*quadlet.ContainerUnit, containerNames []string, name string) ([]runtime.Object, error) {
	var objects []runtime.Object
	labels := map[string]string{
		"app.kubernetes.io/name": name,
	}

	var podContainers []corev1.Container
	var podVolumes []corev1.Volume

	for i, volSpec := range p.Pod.Volume {
		vol, _, err := parseVolumeSpec(volSpec, fmt.Sprintf("pod-vol-%d", i))
		if err != nil {
			return nil, err
		}
		podVolumes = append(podVolumes, *vol)
	}

	for i, c := range containers {
		cName := containerNames[i]
		container, cVolumes, _, err := createContainerSpec(c, cName)
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
	for i, portSpec := range p.Pod.PublishPort {
		_, _, sPort, err := parsePortSpec(portSpec, fmt.Sprintf("pod-port-%d", i))
		if err != nil {
			continue
		}
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

func createContainerSpec(c *quadlet.ContainerUnit, name string) (*corev1.Container, []corev1.Volume, []corev1.ServicePort, error) {
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

	for i, portSpec := range c.Container.PublishPort {
		cPort, _, sPort, err := parsePortSpec(portSpec, fmt.Sprintf("port-%d", i))
		if err != nil {
			continue
		}
		containerPorts = append(containerPorts, *cPort)
		servicePorts = append(servicePorts, *sPort)
	}

	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume

	for i, volSpec := range c.Container.Volume {
		vol, mount, err := parseVolumeSpec(volSpec, fmt.Sprintf("vol-%d", i))
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
            // HealthCmd is usually a command string.
            // Split it? Or pass to sh -c?
            // Podman HealthCmd: "CMD-SHELL curl -f http://localhost/ || exit 1" or just "curl ..."
            // K8s ExecAction command is []string.
            // If we split nicely:
            probeCmd, err := SplitArgs(c.Container.HealthCmd)
            if err == nil {
                livenessProbe = &corev1.Probe{
                    ProbeHandler: corev1.ProbeHandler{
                        Exec: &corev1.ExecAction{
                            Command: probeCmd,
                        },
                    },
                }

                if c.Container.HealthInterval != "" {
                    if d, err := time.ParseDuration(c.Container.HealthInterval); err == nil {
                        livenessProbe.PeriodSeconds = int32(d.Seconds())
                    }
                }
                if c.Container.HealthTimeout != "" {
                    if d, err := time.ParseDuration(c.Container.HealthTimeout); err == nil {
                        livenessProbe.TimeoutSeconds = int32(d.Seconds())
                    }
                }
                if c.Container.HealthStartPeriod != "" {
                    if d, err := time.ParseDuration(c.Container.HealthStartPeriod); err == nil {
                        livenessProbe.InitialDelaySeconds = int32(d.Seconds())
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
    }

    // Resources
    resources := corev1.ResourceRequirements{}
    if c.Container.Memory != "" {
        // Memory limit.
        if q, err := resource.ParseQuantity(c.Container.Memory); err == nil {
            resources.Limits = corev1.ResourceList{corev1.ResourceMemory: q}
            resources.Requests = corev1.ResourceList{corev1.ResourceMemory: q}
        }
    }

    // SecurityContext
    securityContext := &corev1.SecurityContext{}
    hasSecurityContext := false

    if c.Container.User != "" {
        // Try to parse int
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
        nnp := false // AllowPrivilegeEscalation: false means NoNewPrivileges
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

    // Only set if we populated something
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
		ContainerPort: int32(cPort), // nosec G109 - validated above (0-65535)
		Protocol:      corev1.ProtocolTCP,
	}

	sp := &corev1.ServicePort{
		Name:       name,
		Port:       int32(hPort), // nosec G109 - validated above (0-65535)
		TargetPort: intstr.FromInt(cPort),
		Protocol:   corev1.ProtocolTCP,
	}

	return cp, hPort, sp, nil
}

func parseVolumeSpec(spec string, name string) (*corev1.Volume, *corev1.VolumeMount, error) {
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
			claimName = strings.TrimSuffix(source, ".volume")
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
