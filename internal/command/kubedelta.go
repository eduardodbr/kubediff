package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	k8s "github.com/eduardodbr/kubediff/internal/kubernetes"
	"github.com/eduardodbr/kubediff/internal/resource"
	"github.com/fatih/color"
	"github.com/go-test/deep"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"
)

type kubediff struct {
	contexts                []string
	namespaces              []string
	labels                  []string
	resources               []string
	path                    string
	ignoreContainerRegistry bool
	ignoreNonExistent       bool
	ignoreEnv               []string
}

func Newkubediff() *cobra.Command {
	var kubeconfig string
	kd := &kubediff{}
	command := &cobra.Command{
		Use:   "kubediff",
		Short: "A CLI tool to detect differences between Kubernetes resources",
		Long: `kubediff is a command-line tool that helps with detecting differences between 
		Kubernetes resources in different clusters using a dot path to identity the objects to be compared.`,
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
			if err := kd.findDifferences(cmd.Context(), clients, printGenericDifferences); err != nil {
				log.Fatal(err)
			}
		},
	}

	command.Flags().StringSliceVarP(&kd.contexts, "contexts", "c", []string{}, "List of contexts (mandatory)")
	command.Flags().StringVarP(&kd.path, "path", "p", "", "Dot path to the field to compare (mandatory)")
	command.Flags().StringSliceVarP(&kd.resources, "resources", "r", []string{""}, "List of resources to detect changes (mandatory)")
	command.Flags().StringSliceVarP(&kd.namespaces, "namespaces", "n", []string{""}, "List of namespaces (optional)")
	command.Flags().StringSliceVarP(&kd.labels, "labels", "l", []string{}, "List of labels to filter resources (optional)")
	command.Flags().StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig(), "Path to the kubeconfig file (optional, uses $HOME/.kube/config by default)")
	command.Flags().BoolVar(&kd.ignoreNonExistent, "ignore-non-existent", false, "Ignore comparison when resource do not exist in one of the contexts (optional)")
	command.MarkFlagRequired("contexts")
	command.MarkFlagRequired("path")
	command.MarkFlagRequired("resources")
	return command
}

func defaultKubeconfig() string {
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	log.Fatal("Error: kubeconfig not set")
	return ""
}

// printDifferencesFn is a function that prints the differences between the resources in the contexts
// m is a map where the key is the resource name and the value is a map where the key is the context and the value is a slice of values being compared
type printDifferencesFn func(kd *kubediff, namespace string, m map[string]map[string][]any)

// findDifferences finds the differences between the resources in the contexts and prints them using printDiffFunc
func (kd *kubediff) findDifferences(ctx context.Context, clients map[string]*kubernetes.Clientset, printDifferences printDifferencesFn) error {
	for _, namespace := range kd.namespaces {
		for _, resourceType := range kd.resources {
			log.WithField("namespace", namespace).WithField("resource", resourceType).WithField("path", kd.path).Infoln("Finding differences ...")
			g, ctx := errgroup.WithContext(ctx)
			m := make(map[string]map[string][]any)
			lock := sync.Mutex{}
			for context, client := range clients {
				context, namespace, client := context, namespace, client // https://golang.org/doc/faq#closures_and_goroutines
				g.Go(func() error {
					funcToApply := func(item any, name string) error {
						vals, err := getFieldValue(item, kd.path)
						if err != nil {
							return logAndReturnErr(namespace, context, resourceType, fmt.Sprintf("Error: failed to get value for path %s: %v", kd.path, err))
						}
						switch vals.(type) {
						case []interface{}:
							slices, ok := vals.([]interface{})
							if !ok {
								return logAndReturnErr(namespace, context, resourceType, "Error: failed to convert to interface slice")
							}
							for _, slice := range slices {
								addToMap(&lock, m, name, context, slice)
							}
						default:
							addToMap(&lock, m, name, context, vals)
						}
						return nil
					}

					err := resource.Apply(ctx, client, resourceType, kd.labels, namespace, funcToApply)
					if err != nil {
						return logAndReturnErr(namespace, context, resourceType, fmt.Sprintf("Error: failed to apply func to resource %s: %v", resourceType, err))
					}

					return nil
				})
			}
			if err := g.Wait(); err != nil {
				log.Fatal(err)
				return nil
			}

			printDifferences(kd, namespace, m)
		}
	}
	return nil
}

func logAndReturnErr(namespace, context, resourceType, message string) error {
	log.
		WithField("context", context).
		WithField("namespace", namespace).
		WithField("resource", resourceType).
		Error(message)
	return fmt.Errorf(message)
}

func printGenericDifferences(kd *kubediff, namespace string, m map[string]map[string][]any) {
	hasDiff := false
	for resourceName, contextsMap := range m {
		var header strings.Builder
		var diffStr strings.Builder
		resourceDiff := false
		for i, context := range kd.contexts {
			source, ok := contextsMap[context]
			if !ok {
				if !kd.ignoreNonExistent {
					hasDiff = true
					resourceDiff = true
					header.WriteString(color.RedString(fmt.Sprintf("\tNot found in %s\n", context)))
				}
				continue
			}

			// compare current context with all other contexts
			for j := i + 1; j < len(kd.contexts); j++ {
				targetContext := kd.contexts[j]
				target, ok := contextsMap[targetContext]
				if !ok {
					continue
				}

				if len(source) != len(target) {
					hasDiff = true
					resourceDiff = true
					header.WriteString(color.RedString(fmt.Sprintf("\tDifferent number of elements in %s (%d) and %s (%d)\n", context, len(source), targetContext, len(target))))
					continue
				}

				// Compare the two values
				diffs := deep.Equal(source, target)
				if diffs == nil {
					continue
				}

				hasDiff = true
				resourceDiff = true
				diffStr.WriteString(fmt.Sprintf("\tDifference between %s and %s:\n\n", context, targetContext))
				for _, diff := range diffs {
					diffStr.WriteString(fmt.Sprintf("\t\t%v\n", diff))
				}
				diffStr.WriteString("\n")
			}
		}
		if resourceDiff {
			log.Warnf("Found differences for %s", color.HiYellowString(resourceName))
			if str := header.String(); str != "" {
				fmt.Printf("%s\n", str)
			}
			fmt.Printf("%s\n", diffStr.String())
		}
	}
	if !hasDiff {
		log.Info(color.GreenString("No differences found"))
	}
}

func getFieldValue(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	val := reflect.ValueOf(data)

	for i, part := range parts {
		// Dereference pointer if necessary
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		if strings.HasSuffix(part, "[*]") {
			// Handle the slice iteration
			fieldName := strings.TrimSuffix(part, "[*]")
			val = val.FieldByName(fieldName)
			if val.Kind() != reflect.Slice {
				return nil, fmt.Errorf("field '%s' is not a slice", fieldName)
			}

			// Handle the remaining path for each slice element
			remainingPath := strings.Join(parts[i+1:], ".")
			results := []interface{}{}
			for j := 0; j < val.Len(); j++ {
				item := val.Index(j).Interface()
				if remainingPath != "" {
					// Recursively process the remaining path
					result, err := getFieldValue(item, remainingPath)
					if err != nil {
						return nil, err
					}
					results = append(results, result)
				} else {
					results = append(results, item)
				}
			}
			return results, nil
		} else {
			// Normal struct field access
			val = val.FieldByName(part)
			if !val.IsValid() {
				return nil, fmt.Errorf("field '%s' not found", part)
			}
		}
	}

	return val.Interface(), nil
}

// addToMap is a helper function that adds a value to a map m with key resourceName and context
func addToMap(lock *sync.Mutex, m map[string]map[string][]any, resourceName, context string, val any) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := m[resourceName]; !ok {
		m[resourceName] = make(map[string][]any)
	}
	m[resourceName][context] = append(m[resourceName][context], val)
}
