[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/eks-auto-pod-id-assoc/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/eks-auto-pod-id-assoc)](https://goreportcard.com/report/github.com/udhos/eks-auto-pod-id-assoc)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/eks-auto-pod-id-assoc.svg)](https://pkg.go.dev/github.com/udhos/eks-auto-pod-id-assoc)
[![Docker Pulls](https://img.shields.io/docker/pulls/udhos/eks-auto-pod-id-assoc)](https://hub.docker.com/r/udhos/eks-auto-pod-id-assoc)

# eks-auto-pod-id-assoc

[eks-auto-pod-id-assoc](https://github.com/udhos/eks-auto-pod-id-assoc) automatically synchronizes EKS Pod Identity Associations from Service Accounts.

* [Building and running](#building-and-running)
* [How it works](#how-it-works)
* [Configuration file](#configuration-file)
* [Regular expressions](#regular-expressions)
* [Environment variables](#environment-variables)
* [Permissions](#permissions)
* [Topologies](#topologies)
  * [Topology example 1: Running within single cluster](#topology-example-1-running-within-single-cluster)
  * [Topology example 2: Running in a server with ~/\.kube/config managing one cluster](#topology-example-2-running-in-a-server-with-kubeconfig-managing-one-cluster)
  * [Topology example 3: Running in a server with AWS credentials managing multiple clusters](#topology-example-3-running-in-a-server-with-aws-credentials-managing-multiple-clusters)
* [Docker hub](#docker-hub)
* [References](#references)

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

Running docker image:

```bash
docker run --rm -v ./config.yaml:/config.yaml udhos/eks-auto-pod-id-assoc:latest
```

# How it works

`eks-auto-pod-id-assoc` automatically synchronizes Associations from Service Accounts:

- If a ServiceAccount is created with annotation `eks.amazonaws.com/role-arn`, an Association is also created.

- If a ServiceAccount is deleted or the annotation `eks.amazonaws.com/role-arn` is removed, the Association is removed.

Example of an ServiceAccount that causes the creation of an Association:

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

The configuration file is an YAML document declaring a list of clusters.

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
```

field | description
-- | --
role_arn | The role that must be used to make AWS API calls. If not provided, the default behavior is to use the credentials chain.
region | The region to make AWS API calls.
cluster_name | Regular expression for the cluster name. If you want to specify one specific cluster, anchor it like this: `^my-cluster$`. An empty/undefined `cluster_name` will match ALL clusters in the region. NOTICE: When `self=true`, cluster_name is no longer a regex and must be specified as an exact cluster name.
self | Use `self=false` (default) when the tool must acquire kubernetes credentials directly from each targeted cluster; it will need permission to perform `eks:ListClusters` and `eks:DescribeCluster` on the clusters; this is useful when the tool does not have local credentials (like `~/.kube/config`). Set `self=true` to use local credentials (like `~/.kube/config`) instead of generating kubernetes credentials by querying `DescribeCluster`.
annotation | The annotation used in Service Accounts that must be synced. Default is `eks.amazonaws.com/role-arn`.
exclude_service_accounts | List of service accounts to exclude from synchronization. Fields `name` and `namespace` are regular expressions. Empty/undefined field match anything. Matching for exclusion requires BOTH fields (AND operation). A match removes the Service Account and the Association from synchronization. The tool will skip creation and deletion of Association for a match.
restrict_roles | Define a list of roles that are restricted. A restricted role can only be used by Service Accounts that are allowed under the field `allow`. The tool will ignore a Service Account attempting to use a restricted role without being allowed.

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

namespace: ^$   # this matches only the empty namespace, impossible in kubernetes, so it matches NOTHING

namespace: _^$  # negates the previous rule, so it matches anything
```

# Environment variables

These environment variables are available for customization.

Var | Default | Description
-- | -- | --
CONFIG_FILE | config.yaml | Path to configuration file.
INTERVAL | 1m | Interval between resource discovery.
RUN_ONCE | false | If enabled, the tool executes once and exits.
DRY | true | If enabled, the tool does NOT modify anything on AWS EKS. If disabled, the tool will create and delete Associations on AWS EKS as needed to synchronize with Service Accounts.
ADDR | :8080 | Listen address used for health check and metrics.
HEALTH_PATH | /health | Health check path.
METRICS_PATH | /metrics | Metrics path.
METRICS_NAMESPACE | "" | Metrics namespace.
LATENCY_BUCKETS_SECONDS | ".01, .025, .05, .1, .25, .5, 1, 2.5" | Latency buckets in seconds.

# Permissions

The tool needs these permissions on every cluster it should synchronize.

Permission | Comment
-- | --
`eks:ListClusters` and `eks:DescribeCluster` | When `self=false` (default), the tool uses these API calls to generate kubernetes credentials for the k8s API server.
apiGroups:[""] resources:["serviceaccounts"] verbs:["list"] | Discovery of existing Service Accounts.
`eks:ListPodIdentityAssociations` | Discovery of existing Associations.
`eks:CreatePodIdentityAssociation` and `eks:DeletePodIdentityAssociation` | Calls needed to create/destroy Associations on AWS EKS.

# Topologies

Several topologies are possible. Find some examples below.

## Topology example 1: Running within single cluster

There is one single EKS cluster running on region "us-east-1" and the tool runs on one of its nodes.

Use `self=true` to enable in-cluster behavior and set the exact cluster name with `cluster_name`.

You will need to use some other method to give the POD permission to call AWS APIs in order to create/delete Associations on EKS. That's because the POD will not be able to manage its own Association.
One option is to create the Association manually or using some Infra-as-Code like Terraform.

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

`self=false` (default) will generate credentials for k8s api server using `eks:DescribeCluster`.

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

# Docker hub

We provide some built docker images in Docker hub:

https://hub.docker.com/r/udhos/eks-auto-pod-id-assoc

```bash
docker run --rm -v ./config.yaml:/config.yaml udhos/eks-auto-pod-id-assoc:latest
```

# References

- Create Pod Identity Associations based on annotations on ServiceAccounts
  https://github.com/aws/containers-roadmap/issues/2291