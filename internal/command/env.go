package commands

import (
	"fmt"
	"strings"

	k8s "github.com/eduardodbr/kubediff/internal/kubernetes"
	"github.com/fatih/color"
	"github.com/go-test/deep"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

func NewEnvs() *cobra.Command {
	var kubeconfig string
	kd := &kubediff{
		resources: []string{"deployment", "statefulset", "daemonset"},
	}
	command := &cobra.Command{
		Use:   "envs",
		Short: "Detect different env var values between Kubernetes clusters",
		Long: `A CLI tool to detect differences between env vars in deployments, daemonsets or statefulsets with the same 
		name in Kubernetes clusters`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(kd.contexts) < 2 {
				log.Fatal("Error: at least two contexts are required")
				return
			}
			clients, err := k8s.CreateClients(kubeconfig, kd.contexts)
			if err != nil {
				log.Fatalf("Error: failed to create kubernetes clients: %v", err)
				return
			}
			paths := []string{"Spec.Template.Spec.Containers[*].Env"}
			for _, path := range paths {
				kd.path = path
				if err := kd.findDifferences(cmd.Context(), clients, printEnvDifferences); err != nil {
					log.Fatal(err)
				}
			}

		},
	}

	command.Flags().StringSliceVarP(&kd.contexts, "contexts", "c", []string{}, "List of contexts (mandatory)")
	command.Flags().StringSliceVarP(&kd.namespaces, "namespaces", "n", []string{""}, "List of namespaces (optional)")
	command.Flags().StringSliceVarP(&kd.labels, "labels", "l", []string{}, "List of labels (optional)")
	command.Flags().StringSliceVarP(&kd.ignoreEnv, "ignore-env", "i", []string{}, "List env vars to ignore when comparing values (optional)")
	command.Flags().StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig(), "Path to the kubeconfig file")
	command.Flags().BoolVar(&kd.ignoreNonExistent, "ignore-non-existent", false, "Ignore comparison when resource do not exist in one of the contexts (optional)")
	command.MarkFlagRequired("contexts")
	return command
}

func printEnvDifferences(kd *kubediff, namespace string, m map[string]map[string][]any) {
	hasDiff := false
	for resourceName, contexts := range m {
		if processResourceEnvDifferences(kd, resourceName, contexts) {
			hasDiff = true
		}
	}
	if !hasDiff {
		log.Info(color.GreenString("No differences found"))
	}
}

// processResourceEnvDifferences compares the env vars of a resource (resourceName) in different contexts
func processResourceEnvDifferences(kd *kubediff, resourceName string, contexts map[string][]any) bool {
	var header, diffOutput strings.Builder
	resourceDiff := false

	for i, sourceContext := range kd.contexts {
		sourceContainers, ok := contexts[sourceContext]
		if !ok {
			if !kd.ignoreNonExistent {
				resourceDiff = true
				header.WriteString(color.RedString(fmt.Sprintf("\tNot found in %s\n", sourceContext)))
			}
			continue
		}

		// compare current context with all other contexts
		for j := i + 1; j < len(kd.contexts); j++ {
			targetContext := kd.contexts[j]
			targetContainers, ok := contexts[targetContext]
			if !ok {
				continue
			}

			headerDiff, containersDiff, hasDiff := compareContainers(sourceContext, targetContext, sourceContainers, targetContainers, kd.ignoreEnv)
			if hasDiff {
				resourceDiff = true
				header.WriteString(headerDiff)
				diffOutput.WriteString(containersDiff)
			}
		}
	}
	if resourceDiff {
		log.Warnf("Found differences for %s", color.HiYellowString(resourceName))
		if header.Len() > 0 {
			fmt.Printf("%s\n", header.String())
		}

		fmt.Printf("%s\n", diffOutput.String())
	}

	return resourceDiff
}

func compareContainers(sourceContext, targetContext string, sourceContainers, targetContainers []any, ignoreEnv []string) (string, string, bool) {
	var header, diffStr strings.Builder
	if len(sourceContainers) != len(targetContainers) {
		header.WriteString(color.RedString(fmt.Sprintf("\tDifferent number of containers in %s (%d) and %s (%d)\n", sourceContext, len(sourceContainers), targetContext, len(targetContainers))))
		return header.String(), diffStr.String(), true
	}

	resourceDiff := false
	for k := 0; k < len(sourceContainers); k++ {
		sourceContainerEnvs := sourceContainers[k]
		targetContainerEnvs := targetContainers[k]

		sourceEnvs, ok := sourceContainerEnvs.([]corev1.EnvVar)
		if !ok {
			log.Fatalf("Error: failed to convert source env vars to []corev1.EnvVar")
		}
		targetEnvs, ok := targetContainerEnvs.([]corev1.EnvVar)
		if !ok {
			log.Fatalf("Error: failed to convert target env vars to []corev1.EnvVar")
		}
		sourceEnvMap, targetEnvMap := comparableEnvsMap(sourceEnvs, targetEnvs, ignoreEnv)
		diffs := deep.Equal(sourceEnvMap, targetEnvMap)
		if diffs == nil {
			continue
		}
		resourceDiff = true
		diffStr.WriteString(fmt.Sprintf("\tDifferences between %s and %s:\n\n", sourceContext, targetContext))
		for _, diff := range diffs {
			diffStr.WriteString(fmt.Sprintf("\t\t%s\n", diff))
		}
		diffStr.WriteString("\n")
	}
	return header.String(), diffStr.String(), resourceDiff
}

// comparableEnvsMap creates two maps to be compared where the key is the env var name and the value is the env var value
func comparableEnvsMap(source, target []corev1.EnvVar, ignore []string) (sourceMap map[string]any, targetMap map[string]any) {
	// Create maps for source and target
	sourceMap = make(map[string]any)
	targetMap = make(map[string]any)

	// Populate the maps with Name as key and Value as value
	for _, env := range source {
		if !stringInSlice(env.Name, ignore) {
			sourceMap[env.Name] = env.Value
			if env.ValueFrom != nil && env.Name == "" {
				sourceMap[env.Name] = env.ValueFrom
			}
		}
	}
	for _, env := range target {
		if !stringInSlice(env.Name, ignore) {
			targetMap[env.Name] = env.Value
			if env.ValueFrom != nil && env.Name == "" {
				targetMap[env.Name] = env.ValueFrom
			}
		}
	}

	return sourceMap, targetMap
}

func stringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
