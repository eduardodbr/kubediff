# Kubediff

Kubediff is a powerful command-line tool designed to detect differences between Kubernetes resources across multiple clusters. Using a flexible dot-path syntax, it enables you to compare specific fields within Kubernetes objects efficiently.

> ⚠️ **Note**: Kubediff is a pet project and has not been extensively tested. It may contain bugs or unexpected behavior. Use it at your own risk, and feel free to report any issues you encounter!

With Kubediff, you can:

- Compare container images, environment variables, and other fields.
- Filter resources by namespaces, labels, and resource types.
- Ignore missing resources or irrelevant fields for a cleaner comparison.

## Install

To install Kubediff locally, clone the repository, build the binary, and move it to a location in your $PATH:

```bash
git clone https://github.com/eduardodbr/kubediff
cd kubediff
go build -o kubediff cmd/kubediff/main.go
sudo mv kubediff /usr/local/bin/
```

Verify the installation:

```bash
kubediff --help
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
```bash
kubediff --contexts staging,production \
         --resources deployment,statefulset \
         --namespaces monitoring \
         --path "Spec.Template.Spec.Containers[*].Image"
```

### Find different deployment labels filtered by label
```bash
kubediff -c staging,production \
         -r deployment \
         -n monitoring \
         --path "Labels" \
         -l app=prometheus
```

### Find different deployment labels in deployments that exist in both environments

```sh
kubediff -c dev,staging,production \
         -r deployment \
         -n monitoring \
         --path "Labels" \
         --ignore-non-existent
```

## Supported resources

Currently supported Kubernetes resources:

- Deployment
- Daemonset
- Statefulset
- Configmap

Additional resources will be supported soon!

## Extra Commands

Kubediff's generic engine enables comparisons across any Kubernetes resource using dot-paths. However, for certain use cases, Kubediff provides opinionated commands that deliver better insights and formatted output.

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

```bash
kubediff images -c staging,production \
                -n data \
                --ignore-container-registry
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

```bash
kubediff envs -c staging,production \
              -n data \ 
              --ignore-env ENVIRONMENT,REGION
```
