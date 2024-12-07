package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	k8s "github.com/eduardodbr/kubediff/internal/kubernetes"
	"github.com/fatih/color"
	"github.com/go-test/deep"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewImages() *cobra.Command {
	var kubeconfig string
	kd := &kubediff{
		resources: []string{"deployment", "statefulset", "daemonset"},
	}
	command := &cobra.Command{
		Use:   "images",
		Short: "Detect different image tags between Kubernetes clusters",
		Long:  "A CLI tool to detect differences between image tags in deployments with the same name in Kubernetes clusters",
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
			paths := []string{"Spec.Template.Spec.Containers[*].Image", "Spec.Template.Spec.InitContainers[*].Image"}
			for _, path := range paths {
				kd.path = path
				if err := kd.findDifferences(cmd.Context(), clients, printImagesDifferences); err != nil {
					log.Fatal(err)
				}
			}

		},
	}

	command.Flags().StringSliceVarP(&kd.contexts, "contexts", "c", []string{}, "List of contexts (mandatory)")
	command.Flags().StringSliceVarP(&kd.namespaces, "namespaces", "n", []string{""}, "List of namespaces (optional)")
	command.Flags().StringSliceVarP(&kd.labels, "labels", "l", []string{}, "List of labels (optional)")
	command.Flags().BoolVar(&kd.ignoreContainerRegistry, "ignore-container-registry", false, "Ignore container registry in image tags (optional)")
	command.Flags().StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig(), "Path to the kubeconfig file  (optional, uses $HOME/.kube/config by default)")
	command.Flags().BoolVar(&kd.ignoreNonExistent, "ignore-non-existent", false, "Ignore comparison when resource do not exist in one of the contexts (optional)")
	command.MarkFlagRequired("contexts")
	return command
}

func printImagesDifferences(kd *kubediff, namespace string, m map[string]map[string][]any) {
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	var header strings.Builder
	header.WriteString("Service\tNamespace\t")
	for _, context := range kd.contexts {
		header.WriteString(context)
		header.WriteString("\t")
	}
	found := false
	var sb strings.Builder
	for resourceName, contextsMap := range m {
		if (!allElementsExist(kd.contexts, contextsMap) && !kd.ignoreNonExistent) || !kd.allValuesAreEqual(contextsMap) {
			sb.WriteString(fmt.Sprintf("%s\t%s\t", resourceName, namespace))
			for _, context := range kd.contexts {
				sb.WriteString(fmt.Sprintf("%s\t", contextsMap[context]))
			}
			sb.WriteString("\n")
			found = true
		}
	}
	if !found {
		log.Infoln(color.GreenString("No differences found"))
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, header.String())
	fmt.Fprintln(w, sb.String())
	w.Flush()
	return
}

func allElementsExist(elements []string, m map[string][]any) bool {
	for _, element := range elements {
		if _, ok := m[element]; !ok {
			return false
		}
	}
	return true
}

func (kd *kubediff) allValuesAreEqual(m map[string][]any) bool {
	if len(m) == 0 {
		return true
	}
	var target []any
	for _, source := range m {
		if kd.ignoreContainerRegistry {
			for i, value := range source {
				str, ok := value.(string)
				if !ok {
					return false
				}
				source[i] = removeContainerRegistry(str)
			}
		}
		if target == nil {
			target = source
		} else if diffs := deep.Equal(source, target); len(diffs) > 0 {
			return false
		}
	}
	return true
}

func removeContainerRegistry(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) < 2 {
		return image
	}
	return strings.Join(parts[1:], "/")
}
