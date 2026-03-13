[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/eks-auto-pod-id-assoc/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/eks-auto-pod-id-assoc)](https://goreportcard.com/report/github.com/udhos/eks-auto-pod-id-assoc)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/eks-auto-pod-id-assoc.svg)](https://pkg.go.dev/github.com/udhos/eks-auto-pod-id-assoc)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/eks-auto-pod-id-assoc)](https://artifacthub.io/packages/search?repo=eks-auto-pod-id-assoc)
[![Docker Pulls](https://img.shields.io/docker/pulls/udhos/eks-auto-pod-id-assoc)](https://hub.docker.com/r/udhos/eks-auto-pod-id-assoc)

# eks-auto-pod-id-assoc

[eks-auto-pod-id-assoc](https://github.com/udhos/eks-auto-pod-id-assoc) automatically synchronizes EKS Pod Identity Associations from Service Accounts.

* [Why](#why)
* [Building and running](#building-and-running)
* [Quick Start](#quick-start)
  * [Prerequisites](#prerequisites)
  * [Installation](#installation)
  * [Configuration](#configuration)
  * [Run](#run)
  * [Verify](#verify)
* [How it works](#how-it-works)
* [Configuration file](#configuration-file)
* [Regular expressions](#regular-expressions)
* [Environment variables](#environment-variables)
* [Permissions](#permissions)
* [Topologies](#topologies)
  * [Topology example 1: Running within single cluster](#topology-example-1-running-within-single-cluster)
  * [Topology example 2: Running in a server with ~/\.kube/config managing one cluster](#topology-example-2-running-in-a-server-with-kubeconfig-managing-one-cluster)
  * [Topology example 3: Running in a server with AWS credentials managing multiple clusters](#topology-example-3-running-in-a-server-with-aws-credentials-managing-multiple-clusters)
* [Metrics](#metrics)
* [Docker Hub](#docker-hub)
* [Helm chart](#helm-chart)
  * [Using the helm repository](#using-the-helm-repository)
  * [Using local chart](#using-local-chart)
* [Contributing](#contributing)
* [References](#references)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc.go)

# Why

- **Eliminate manual sync work** - Stop managing Pod Identity Associations in AWS Console; define everything in Kubernetes.
- **Zero drift guarantee** - Kubernetes is the source of truth; no more mismatches between ServiceAccounts and EKS associations.
- **Scale effortlessly** - Manage 1 cluster or multiple with a single config; regex-based discovery scales automatically.
- **Built-in security guardrails** - Declare which roles each ServiceAccount can use; prevent privilege escalation across applications.
- **GitOps ready** - Codify your entire infrastructure as Kubernetes manifests; works with any CI/CD pipeline.
- **No CRDs required** - Just annotate ServiceAccounts, run the tool, and it syncs associations automatically.
- **Ephemeral cluster friendly** - Works well with clusters that are recreated frequently; associations are re-synced automatically whenever the cluster is rebuilt.
- **Flexible deployment** - Run inside the cluster, on a management host, or across multiple AWS accounts without code changes.

# Building and running

Quick build:

```bash
go install github.com/udhos/eks-auto-pod-id-assoc/cmd/eks-auto-pod-id-assoc@latest
```

Full build for development, including linting etc:

```bash
git clone https://github.com/udhos/eks-auto-pod-id-assoc
cd eks-auto-pod-id-assoc
./build.sh
```

Just run the binary:

```bash
eks-auto-pod-id-assoc
```

Running Docker image:

```bash
docker run --rm -v ./config.yaml:/config.yaml udhos/eks-auto-pod-id-assoc:latest
```

# Quick Start

## Prerequisites

- An EKS cluster with Pod Identity enabled
- AWS credentials with permissions to manage Pod Identity Associations (see [Permissions](#permissions))

## Installation

Install via Go:

```bash
go install github.com/udhos/eks-auto-pod-id-assoc/cmd/eks-auto-pod-id-assoc@latest
```

Or use Docker:

```bash
docker pull udhos/eks-auto-pod-id-assoc:latest
```

## Configuration

Create a `config.yaml` file with your cluster details:

```yaml
clusters:
  - region: us-east-1
    cluster_name: my-cluster
    self: true
```

Replace `us-east-1` with your AWS region and `my-cluster` with your EKS cluster name.

## Run

With Go install:

```bash
export DRY=false ;# apply changes

eks-auto-pod-id-assoc
```

With Docker:

```bash
# dry:
docker run --rm -v ./config.yaml:/config.yaml udhos/eks-auto-pod-id-assoc:latest

# apply changes:
docker run --rm -e DRY=false -v ./config.yaml:/config.yaml udhos/eks-auto-pod-id-assoc:latest
```

The tool will start monitoring Service Accounts and, if `DRY=false` (it defaults to `true`), automatically create/delete Pod Identity Associations based on the `eks.amazonaws.com/role-arn` annotation.

For production deployments, consider using the [Helm chart](#helm-chart) or see [Topologies](#topologies) for advanced configurations.

## Verify

```bash
aws eks list-pod-identity-associations --cluster-name my-cluster
```

# How it works

`eks-auto-pod-id-assoc` automatically synchronizes [EKS Pod Identity Associations](https://docs.aws.amazon.com/eks/latest/eksctl/pod-identity-associations.html) from K8s Service Accounts:

- If a ServiceAccount is created with annotation `eks.amazonaws.com/role-arn`, an Association is also created.

- If a ServiceAccount is deleted or the annotation `eks.amazonaws.com/role-arn` is removed, the Association is removed.

In order to react quickly, the tool watches Kubernetes API server for any changes in Service Accounts, and takes action immediately.

However to reconcile changes applied to EKS Pod Identity Associations, the tool queries the EKS API periodically, with configurable period, 1min by default.

Example of a ServiceAccount that causes the creation of an Association:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::111122223333:role/my-role
  name: test
  namespace: default
```

# Configuration file

The configuration file is a YAML document declaring a list of clusters.

Example:

```yaml
clusters:
  - role_arn: arn:aws:iam::123456789012:role/AnotherRole
    region: us-east-1
    cluster_name: ^example-cluster-2$
    self: false
    annotation: eks.amazonaws.com/role-arn
    exclude_service_accounts:
      - name: ^sa1$
        namespace: ^default$
      - namespace: ^kube-system$
    restrict_roles:
      - role_arn: ^arn:aws:iam::123456789012:role/role1$
        allow:
          - name: ^sa3$
            namespace: ^default$
          - name: ^sa4$
            namespace: ^kube-system$
      - role_arn: ^arn:aws:iam::123456789012:role/role2$
        allow:
          - name: ^sa5$
    pod_identity_association_tags:
      managed-by: eks-auto-pod-id-assoc
    max_concurrency: 5
    purge_external_stale_associations: false
```

field | description
-- | --
role_arn | The role that must be used to make AWS API calls. If not provided, the default behavior is to use the credentials chain.
region | The region to make AWS API calls.
cluster_name | Regular expression for the cluster name. If you want to specify one specific cluster, anchor it like this: `^my-cluster$`. An empty/undefined `cluster_name` will match ALL clusters in the region. NOTICE: When `self=true`, cluster_name is no longer a regex and must be specified as an exact cluster name.
self | `self=true` means the tool must get API server credentials from local environment (`~/.kube/config` or in-cluster). Use `self=false` (default) when the tool must acquire Kubernetes credentials directly from each targeted cluster; it will need permission to perform `eks:ListClusters` and `eks:DescribeCluster` on the clusters; this is useful when the tool does not have local credentials (`~/.kube/config` or in-cluster). Set `self=true` to use local credentials (`~/.kube/config` or in-cluster) instead of generating Kubernetes credentials by querying `DescribeCluster`.
annotation | The annotation used in Service Accounts that must be synced. Default is `eks.amazonaws.com/role-arn`.
exclude_service_accounts | List of service accounts to exclude from synchronization. This option is useful if you want to exclude some Service Accounts from auto synchronization because they are managed elsewhere. Fields `name` and `namespace` are regular expressions. Empty/undefined field match anything. Matching for exclusion requires BOTH fields (AND operation). A match removes the Service Account and the Association from synchronization. The tool will skip creation and deletion of Association for a match.
restrict_roles | Define a list of roles that are restricted. A restricted role can only be used by Service Accounts that are allowed under the field `allow`. The tool will ignore a Service Account attempting to use a restricted role without being allowed. The roles are processed in the order listed under `restrict_roles`. Only the first matching role regex is used.
pod_identity_association_tags | Tags added to Associations. Default is `managed-by=eks-auto-pod-id-assoc`. **CAUTION** You can safely change this field before running the tool against a cluster. However once the tool has created associations with tagging in a cluster, you should **NOT** modify the tagging afterwards. If you do change the tagging, the associations created with previous tagging will linger in the cluster and the tool will be unable to delete or to update the old associations. If you did change the tagging in a cluster under synchronization, and now you need to restore the tool operation, you must manually delete all associations lingering with previous tagging.
max_concurrency | Limit concurrency level for EKS API operations (create/delete) over multiple Associations. Default is 5.
purge_external_stale_associations | false | **false**: Only manages associations created and tagged by this tool. **true**: Manages all associations in the cluster. **Warning**: If enabled, any association (including those created by Terraform/Console) that lacks a corresponding Kubernetes Service Account will be deleted. **NOTE**: This flags controls only external associations. The tool always cleans internal self-created associations as soon as they become stale (orphan from Service Account).

# Regular expressions

SYNOPSIS

```yaml
clusters:
  - cluster_name: ^example-cluster-2$                     # <--- regex (only if self=false)
    exclude_service_accounts:
      - name: ^sa1$                                       # <--- regex
        namespace: ^default$                              # <--- regex
    restrict_roles:
      - role_arn: ^arn:aws:iam::123456789012:role/role1$  # <--- regex
        allow:
          - name: ^sa3$                                   # <--- regex
            namespace: ^default$                          # <--- regex
```

Some fields must be given regular expressions.

The full regex syntax is described here: https://github.com/google/re2/wiki/Syntax

If you need exact match, anchor the value like this: `^example$`

The field `cluster_name` is not a regex when the field `self` is set to `true`.

**NEGATION**

You can negate, or invert, a regex by prefixing it with `_` (underscore).

The negation underscore only works as the first character in the expression.
In any other position the underscore is literal.

This negation is a special extension for regex matching that is not available elsewhere.

Example:

```yaml
cluster_name: ^test # matches anything starting with 'test'

cluster_name: _^test # matches anything NOT starting with 'test'
```

**EMPTY EXPRESSIONS**

Empty expressions match *anything*. See some examples below.

```yaml
#namespace: ""  # if you omit a regex field, it becomes empty, so it matches anything

namespace: ""   # an empty regex field matches anything

namespace: _    # this negates the empty regex, thus it matches NOTHING

namespace: ^$   # this matches only the empty namespace, impossible in Kubernetes, so it matches NOTHING

namespace: _^$  # negates the previous rule, so it matches anything
```

# Environment variables

These environment variables are available for customization.

Var | Default | Description
-- | -- | --
LOG_LEVEL | info | Set log level: debug, info, warn, error
LOG_JSON | false | Enable JSON logging.
CONFIG_FILE | config.yaml | Path to configuration file.
INTERVAL | 1m | Interval between reconciliations. In order to react quickly, the tool watches Kubernetes API server for any changes in Service Accounts, and takes action immediately. However to reconcile changes applied to EKS Pod Identity Associations, the tool queries the EKS API periodically.
RUN_ONCE | false | If enabled, the tool executes once and exits.
DRY | true | If enabled, the tool does NOT modify anything on AWS EKS. If disabled, the tool will create and delete Associations on AWS EKS as needed to synchronize with Service Accounts.
ADDR | :8080 | Listen address used for health check and metrics.
HEALTH_PATH | /health | Health check path.
METRICS_PATH | /metrics | Metrics path.
METRICS_NAMESPACE | "" | Metrics namespace.
LATENCY_BUCKETS_SECONDS | ".005, .01, .025, .05, .1, .25, .5, 1, 2.5" | Prometheus histogram latency buckets in seconds.
DOGSTATSD_SAMPLE_RATE | 1.0 | Dogstatsd sample rate.
DOGSTATSD_ENABLE | false | Enable Dogstatsd metrics.
DD_AGENT_HOST | localhost | Dogstatsd agent hostname.
DD_SERVICE | undefined | Set service name for Datadog.

# Permissions

The tool needs these permissions on every cluster it should synchronize.

Permission | Comment
-- | --
K8s RBAC: apiGroups:[""] resources:["serviceaccounts"] verbs:["get","list","watch"] | Discovery of existing Service Accounts.
`eks:ListClusters` and `eks:DescribeCluster` | When `self=false` (default), the tool uses these API calls to generate Kubernetes credentials for the K8s API server.
`eks:ListPodIdentityAssociations` and `resourcegroupstaggingapi:GetResources` | Discovery of existing Associations.
`eks:CreatePodIdentityAssociation` and `eks:DeletePodIdentityAssociation` | Calls needed to create/destroy Associations on AWS EKS.
`iam:PassRole`, `iam:GetRole` and `eks:TagResource` | Permissions required to create Associations on AWS EKS.

See examples:

- K8s clusterrole in [examples/clusterrole_example.yaml](examples/clusterrole_example.yaml).

- K8s clusterrolebinding in [examples/clusterrolebinding_example.yaml](examples/clusterrolebinding_example.yaml).

- AWS policy in [examples/aws_iam_policy_example.json](examples/aws_iam_policy_example.json).

# Topologies

Several topologies are possible. Find some examples below.

## Topology example 1: Running within single cluster

There is one single EKS cluster running on region "us-east-1" and the tool runs on one of its nodes.

Use `self=true` to enable in-cluster behavior and set the exact cluster name with `cluster_name`.

You will need to use some other method to give the Pod permission to call AWS APIs in order to create/delete Associations on EKS. That's because the Pod will not be able to manage its own Association.
One option is to create the Association manually or using some Infrastructure-as-Code like Terraform.

```yaml
clusters:
  - region: us-east-1
    cluster_name: my-cluster # self=true requires exact cluster name
    self: true
```

## Topology example 2: Running in a server with ~/.kube/config managing one cluster

Install the tool on a server properly configured with `~/.kube/config`.

Use `self=true` to enable `~/.kube/config` and set the exact cluster name with `cluster_name`.

You will need to start the tool with the proper AWS credentials to allow AWS calls to EKS APIs.
You could use `~/.aws/config` or `AWS_PROFILE` or credentials in env vars.

```yaml
clusters:
  - #role_arn: arn:aws:iam::123456789012:role/role1 # if AssumeRole is needed
    region: us-east-1
    cluster_name: my-cluster # self=true requires exact cluster name
    self: true
```

## Topology example 3: Running in a server with AWS credentials managing multiple clusters

Install the tool on a server. It does NOT have `~/.kube/config`.

`self=false` (default) will generate credentials for K8s api server using `eks:DescribeCluster`.

With `self=false`, `cluster_name` must be set to a regex. If you want an exact match,
anchor it like this: `cluster_name: ^my-cluster$`

Run the tool with usual AWS credentials (You could use `~/.aws/config` or `AWS_PROFILE` or credentials in env vars.)

```yaml
clusters:

- #role_arn: arn:aws:iam::123456789012:role/AnotherRole # if AssumeRole is needed
  region: sa-east-1
  cluster_name: ^my-cluster$ # exact cluster

- #role_arn: arn:aws:iam::123456789012:role/AnotherRole # if AssumeRole is needed
  region: us-east-1
  cluster_name: ^my- # auto discover all clusters with name my-
```

# Metrics

The metrics are available both for Prometheus and for Datadog Dogstatsd.

If you want to enable the metrics for Datadog/Dogstatsd, configure these env vars:

```bash
DOGSTATSD_ENABLE=true            ;# enable this
DD_AGENT_HOST=datadog.datadog    ;# point to actual agent hostname
DD_SERVICE=eks-auto-pod-id-assoc ;# you can customize this as you want
```


Metric                    | Prometheus | Datadog      | Dimensions             | Comment
-- | -- | -- | -- | --
service_accounts          | gauge      | gauge        | cluster, ignore_reason | Number of service accounts.
pod_identity_associations | gauge      | gauge        | cluster, ignore_reason | Number of associations.
discover_latency_seconds  | gauge      | distribution | cluster                | Discover latency.
reconcile_latency_seconds | gauge      | distribution | cluster                | Reconcile latency.
api_latency_seconds       | histogram  | distribution | cluster, api, status   | API latency.

Possible dimensions values:

- ignore_reason: not_ignored, excluded, restricted_role
- status: ok, error
- api: serviceaccounts.list, eks:ListClusters, eks:DescribeCluster, eks:ListPodIdentityAssociations, eks:CreatePodIdentityAssociation, eks:DeletePodIdentityAssociation, resourcegroupstaggingapi:GetResources

# Docker Hub

We provide some built container images in Docker Hub:

https://hub.docker.com/r/udhos/eks-auto-pod-id-assoc

```bash
docker run --rm -v ./config.yaml:/config.yaml udhos/eks-auto-pod-id-assoc:latest
```

# Helm chart

## Using the helm repository

See: https://udhos.github.io/eks-auto-pod-id-assoc/

## Using local chart

```bash
# render chart to stdout
helm template eks-auto-pod-id-assoc ./charts/eks-auto-pod-id-assoc

# install chart
helm upgrade --install eks-auto-pod-id-assoc ./charts/eks-auto-pod-id-assoc

# logs
kubectl logs deploy/eks-auto-pod-id-assoc -f
```

# Contributing

Contributions are welcome!

- [Discussions](https://github.com/udhos/eks-auto-pod-id-assoc/discussions): For "how-to" questions or architectural brainstorming.

- [Issues](https://github.com/udhos/eks-auto-pod-id-assoc/issues): To report reproducible bugs or request specific features.

- [Pull Requests](https://github.com/udhos/eks-auto-pod-id-assoc/pulls): Please ensure ./build.sh passes (including linting) before submitting.

# References

- Create Pod Identity Associations based on annotations on ServiceAccounts
  https://github.com/aws/containers-roadmap/issues/2291