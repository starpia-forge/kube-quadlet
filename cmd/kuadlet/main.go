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

var (
	outputOneFile bool
	splitOutput   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kuadlet",
		Short: "Convert Podman Quadlet files to Kubernetes manifests",
	}

	convertCmd := &cobra.Command{
		Use:   "convert [file or directory]...",
		Short: "Convert Quadlet files to Kubernetes YAML",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runConvert,
	}

	convertCmd.Flags().BoolVar(&outputOneFile, "one-file", true, "Output all manifests to stdout separated by '---' (default)")
	convertCmd.Flags().BoolVar(&splitOutput, "split", false, "Write manifests to separate files in current directory (overrides --one-file)")

	rootCmd.AddCommand(convertCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runConvert(cmd *cobra.Command, args []string) error {
	var inputFiles []string

	// Recursive directory walk or just list files
	for _, arg := range args {
		path := filepath.Clean(arg)
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to access %s: %w", path, err)
		}

		if info.IsDir() {
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					ext := filepath.Ext(p)
					if isSupportedExtension(ext) {
						inputFiles = append(inputFiles, p)
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to walk directory %s: %w", path, err)
			}
		} else {
			if isSupportedExtension(filepath.Ext(path)) {
				inputFiles = append(inputFiles, path)
			}
		}
	}

	if len(inputFiles) == 0 {
		return fmt.Errorf("no supported Quadlet files found")
	}

	// Process files
	type result struct {
		Name    string
		Objects []runtime.Object
		Path    string
	}

	var results []result
	processedNames := make(map[string]string) // name -> path

	for _, inputFile := range inputFiles {
		absPath, err := filepath.Abs(inputFile)
		if err != nil {
			return err
		}
		dir := filepath.Dir(absPath)
		filename := filepath.Base(absPath)
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)

		// Check collision
		if existingPath, ok := processedNames[name]; ok {
			safeName := sanitize(name)
			safeExisting := sanitize(existingPath)
			safeCurrent := sanitize(absPath)
			// #nosec G705
			fmt.Fprintf(os.Stderr, "Warning: Name collision detected for resource '%s'. Defined in:\n  - %s\n  - %s\nProceeding with conversion, but this may cause conflicts in output.\n", safeName, safeExisting, safeCurrent)
		}
		processedNames[name] = absPath

		f, err := os.Open(inputFile)
		if err != nil {
			safePath := sanitize(inputFile)
			safeErr := sanitize(err.Error())
			fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %s\n", safePath, safeErr) // #nosec G705
			continue
		}

		u, err := parser.Parse(f)
		_ = f.Close() // Close immediately after parsing
		if err != nil {
			safePath := sanitize(inputFile)
			safeErr := sanitize(err.Error())
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %s\n", safePath, safeErr) // #nosec G705
			continue
		}

		var objects []runtime.Object

		switch ext {
		case ".container":
			c := quadlet.LoadContainer(u)
			if c.Container.Pod != "" {
				// Check if we are processing the pod too?
				// Actually, if we are processing all files, the pod logic in ConvertPod handles its containers.
				// If we process this container individually here, we get a Deployment.
				// If ConvertPod handles it, we get it inside the Pod's Deployment.
				// We should probably skip standalone conversion if it belongs to a Pod AND we are processing that Pod?
				// But we don't know if we are processing that Pod yet (might be later in list).
				// For now, let's just warn and convert as standalone (wrapper logic not applied).
				// This is consistent with previous behavior.
				safeFilename := sanitize(filename)
				safePod := sanitize(c.Container.Pod)
				fmt.Fprintf(os.Stderr, "Warning: Container %s belongs to pod %s. Converting as standalone Deployment (pod wrapper logic not applied).\n", safeFilename, safePod) // #nosec G705
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
			// Note: This still scans the directory of the pod file for containers.
			// It assumes containers are in the same directory.
			containers, containerNames, err := findContainersForPod(dir, filename)
			if err != nil {
				return fmt.Errorf("failed to scan for containers: %w", err)
			}
			objs, err := converter.ConvertPod(p, containers, containerNames, name)
			if err != nil {
				return err
			}
			objects = objs

		case ".kube":
			k := quadlet.LoadKube(u)
			objs, err := converter.ConvertKube(k, name)
			if err != nil {
				return err
			}
			objects = objs

		case ".network":
			n := quadlet.LoadNetwork(u)
			objs, err := converter.ConvertNetwork(n, name)
			if err != nil {
				return err
			}
			objects = objs

		case ".image":
			i := quadlet.LoadImage(u)
			objs, err := converter.ConvertImage(i, name)
			if err != nil {
				return err
			}
			objects = objs

		case ".build":
			b := quadlet.LoadBuild(u)
			objs, err := converter.ConvertBuild(b, name)
			if err != nil {
				return err
			}
			objects = objs
		case ".artifact":
			a := quadlet.LoadArtifact(u)
			objs, err := converter.ConvertArtifact(a, name)
			if err != nil {
				return err
			}
			objects = objs
		}

		if len(objects) > 0 {
			results = append(results, result{Name: name, Objects: objects, Path: absPath})
		}
	}

	// Output
	first := true
	for _, res := range results {
		if splitOutput {
			// Write to file
			outFilename := fmt.Sprintf("%s.yaml", res.Name)
			f, err := os.Create(outFilename)
			if err != nil {
				return fmt.Errorf("failed to create output file %s: %w", outFilename, err)
			}

			for i, obj := range res.Objects {
				if i > 0 {
					f.WriteString("---\n")
				}
				data, err := yaml.Marshal(obj)
				if err != nil {
					f.Close()
					return fmt.Errorf("failed to marshal object: %w", err)
				}
				f.Write(data)
			}
			f.Close()
		} else {
			// Stdout
			if !first {
				fmt.Println("---")
			}
			first = false

			for i, obj := range res.Objects {
				if i > 0 {
					fmt.Println("---")
				}
				data, err := yaml.Marshal(obj)
				if err != nil {
					return fmt.Errorf("failed to marshal object: %w", err)
				}
				fmt.Print(string(data))
			}
		}
	}

	return nil
}

func isSupportedExtension(ext string) bool {
	switch ext {
	case ".container", ".volume", ".pod", ".kube", ".network", ".image", ".build", ".artifact":
		return true
	}
	return false
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
