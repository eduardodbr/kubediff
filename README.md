# Kubediff

Kubediff is a command-line tool that detects differences between Kubernetes resources in different clusters using a dot path to identity the objects to be compared. 

## Install

```
git clone https://github.com/eduardodbr/kubediff
go build -o kubediff cmd/kubediff/main.go
mv kubediff /usr/local/bin/
```

## Usage 
```
Usage:
  kubediff [flags]
  kubediff [command]

Available Commands:
  envs        Detect different env var values between Kubernetes clusters
  help        Help about any command
  images      Detect different image tags between Kubernetes clusters

Flags:
  -c, --contexts strings      List of contexts (mandatory)
  -h, --help                  help for kubediff
      --ignore-non-existent   Ignore comparison when resource do not exist in one of the contexts (optional)
      --kubeconfig string     Path to the kubeconfig file (default "$HOME/.kube/config")
  -l, --labels strings        List of labels to filter resources (optional)
  -n, --namespaces strings    List of namespaces (optional)
  -p, --path string           Dot path to the field to compare (mandatory)
  -r, --resources strings     List of resources to detect changes (mandatory)
```

## Examples

### Find different container images in deployments and statefulsets
```
kubediff --contexts staging,production --resources deployment,statefulset --namespaces monitoring --path "Spec.Template.Spec.Containers[*].Image"
```

### Find different deployment labels filtered by label
```
kubediff -c staging,production -r deployment -n monitoring --path "Labels" -l app=prometheus
```

### Find different deployment labels in deployments that exist in both environments
```
kubediff -c dev,staging,production -r deployment -n monitoring --path "Labels" --ignore-non-existent
```

## Supported resources (more soon)

- Deployment
- Daemonset
- Statefulset
- Configmap

## Extra Commands

Kubediff finds differences between resources using a single path and a generic approach. The following commands use the generic kubediff engine but parses the output in an opinionated way to provide better insights. 

## Images

The `images` command finds differences in `Deployment`, `StatefulSet` and `DaemonSets` in both `"Spec.Template.Spec.Containers[*].Image` and `Spec.Template.Spec.InitContainers[*].Image` paths and provides the output in a table format for easier inspection.

### Usage 

```
Usage:
  kubediff images [flags]

Flags:
  -c, --contexts strings            List of contexts (mandatory)
  -h, --help                        help for images
      --ignore-container-registry   Ignore container registry in image tags (optional)
      --ignore-non-existent         Ignore comparison when resource do not exist in one of the contexts (optional)
      --kubeconfig string           Path to the kubeconfig file (default "$HOME/.kube/config")
  -l, --labels strings              List of labels (optional)
  -n, --namespaces strings          List of namespaces (optional)
```

### Examples

#### Find the difference between images ignoring the container registry

```
kubediff images -c staging,production -n data --ignore-container-registry
```

## Envs

The `envs` command finds differences in `Deployment`, `StatefulSet` and `DaemonSets` in path `Spec.Template.Spec.Containers[*].Env` and compares the env var values by name.

### Usage

```
Usage:
  kubediff envs [flags]

Flags:
  -c, --contexts strings      List of contexts (mandatory)
  -h, --help                  help for envs
  -i, --ignore-env strings    List env vars to ignore when comparing values (optional)
      --ignore-non-existent   Ignore comparison when resource do not exist in one of the contexts (optional)
      --kubeconfig string     Path to the kubeconfig file (default "$HOME/.kube/config")
  -l, --labels strings        List of labels (optional)
  -n, --namespaces strings    List of namespaces (optional)
```

### Examples

#### Find the difference between env vars ignoring some envs

```
kubediff envs -c staging,production -n data --ignore-env ENVIRONMENT,REGION
```
