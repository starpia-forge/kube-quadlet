package main

import (
	"fmt"
	"kuadlet/pkg/converter"
	"kuadlet/pkg/parser"
	"kuadlet/pkg/quadlet"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kuadlet",
		Short: "Convert Podman Quadlet files to Kubernetes manifests",
	}

	convertCmd := &cobra.Command{
		Use:   "convert [file]",
		Short: "Convert a Quadlet file to Kubernetes YAML",
		Args:  cobra.ExactArgs(1),
		RunE:  runConvert,
	}

	rootCmd.AddCommand(convertCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runConvert(cmd *cobra.Command, args []string) error {
	inputFile := filepath.Clean(args[0])
	absPath, err := filepath.Abs(inputFile)
	if err != nil {
		return err
	}
	dir := filepath.Dir(absPath)
	filename := filepath.Base(absPath)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			safeInput := sanitize(inputFile)
			safeErr := sanitize(closeErr.Error())
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %s\n", safeInput, safeErr) // #nosec G705
		}
	}()

	u, err := parser.Parse(f)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	var objects []runtime.Object

	switch ext {
	case ".container":
		c := quadlet.LoadContainer(u)
		// Check if it belongs to a pod
		if c.Container.Pod != "" {
			// Sanitize output for XSS/log injection prevention (G705)
			safeFilename := sanitize(filename)
			safePod := sanitize(c.Container.Pod)
			// #nosec G705
			fmt.Fprintf(os.Stderr, "Warning: Container %s belongs to pod %s. Converting as standalone Deployment (pod wrapper logic not applied).\n", safeFilename, safePod)
		}
		objs, err := converter.ConvertContainer(c, name)
		if err != nil {
			return err
		}
		objects = objs

	case ".volume":
		v := quadlet.LoadVolume(u)
		objs, err := converter.ConvertVolume(v, name)
		if err != nil {
			return err
		}
		objects = objs

	case ".pod":
		p := quadlet.LoadPod(u)
		// Find containers
		containers, containerNames, err := findContainersForPod(dir, filename)
		if err != nil {
			return fmt.Errorf("failed to scan for containers: %w", err)
		}
		objs, err := converter.ConvertPod(p, containers, containerNames, name)
		if err != nil {
			return err
		}
		objects = objs

	default:
		return fmt.Errorf("unsupported file extension: %s", ext)
	}

	// Output YAML
	for i, obj := range objects {
		if i > 0 {
			fmt.Println("---")
		}

		// Use sigs.k8s.io/yaml to marshal k8s objects
		// Marshal handles JSON tags correctly
		data, err := yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf("failed to marshal object: %w", err)
		}
		fmt.Print(string(data))
	}

	return nil
}

func findContainersForPod(dir string, podFilename string) ([]*quadlet.ContainerUnit, []string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	var containers []*quadlet.ContainerUnit
	var names []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) != ".container" {
			continue
		}

		path := filepath.Clean(filepath.Join(dir, file.Name()))
		f, err := os.Open(path)
		if err != nil {
			// Warn and skip?
			safePath := sanitize(path)
			safeErr := sanitize(err.Error())
			fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %s\n", safePath, safeErr)
			continue
		}

		// Parse
		u, err := parser.Parse(f)
		_ = f.Close()
		if err != nil {
			continue
		}

		// Load minimal to check Pod
		// Or just load full? Full is fine.
		c := quadlet.LoadContainer(u)

		if c.Container.Pod == podFilename || c.Container.Pod == strings.TrimSuffix(podFilename, ".pod") {
			containers = append(containers, c)
			names = append(names, strings.TrimSuffix(file.Name(), ".container"))
		}
	}
	return containers, names, nil
}

func sanitize(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "\r", "")
}
