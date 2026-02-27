[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/eks-auto-pod-id-assoc/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/eks-auto-pod-id-assoc)](https://goreportcard.com/report/github.com/udhos/eks-auto-pod-id-assoc)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/eks-auto-pod-id-assoc.svg)](https://pkg.go.dev/github.com/udhos/eks-auto-pod-id-assoc)
[![Docker Pulls](https://img.shields.io/docker/pulls/udhos/eks-auto-pod-id-assoc)](https://hub.docker.com/r/udhos/eks-auto-pod-id-assoc)

# eks-auto-pod-id-assoc

[eks-auto-pod-id-assoc](https://github.com/udhos/eks-auto-pod-id-assoc) automatically synchronizes EKS Pod Identity Associations from Service Accounts.

* [Building](#building)
* [How it works](#how-it-works)
* [Configuration file](#configuration-file)
* [Environment variables](#environment-variables)
* [Permissions](#permissions)
* [Docker hub](#docker-hub)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc.go)

# Building

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
    exclude_namespaces:
      - kube-system
```

field | description
-- | --
role_arn | The role that must be used to make AWS API calls. If not provided, the default behavior is to use the credentials chain.
region | The region to make AWS API calls.
cluster_name | Regular expression for the cluster name. If you want to specify one specific cluster, anchor it like this: `^my-cluster$`. An empty/undefined `cluster_name` will match ALL clusters in the region. NOTICE: When `self=true`, cluster_name is no longer a regex and must be specified as an exact cluster name.
self | Use `self=false` (default) when the tool must acquire kubernetes credentials directly from each targeted cluster; it will need permission to perform `eks:ListClusters` and `eks:DescribeCluster` on the clusters; this is useful when the tool does not have local credentials (like `~/.kube/config`). Set `self=true` to use local credentials (like `~/.kube/config`) instead of generating kubernetes credentials by querying `DescribeCluster`.
annotation | The annotation used in Service Accounts that must be synced. Default is `eks.amazonaws.com/role-arn`.
exclude_namespaces | List of namespaces to exclude from synchronization.

# Environment variables

These environment variables are available for customization.

Var | Default | Description
-- | -- | --
CONFIG_FILE | config.yaml | Path to configuration file.
INTERVAL | 1m | Interval between resource discovery.
RUN_ONCE | false | If enabled, the tool executes once and exits.
DRY | true | If enabled, the tool does NOT modify anything on AWS EKS. If disabled, the tool will create and delete Associations on AWS EKS as needed to synchronize with Service Accounts.

# Permissions

The tool needs these permissions on every cluster it should synchronize.

Permission | Comment
-- | --
`eks:ListClusters` and `eks:DescribeCluster` | When `self=false` (default), the tool uses these API calls to generate kubernetes credentials for the k8s API server.
apiGroups:[""] resources:["serviceaccounts"] verbs:["list"] | Discovery of existing Service Accounts.
`eks:ListPodIdentityAssociations` | Discovery of existing Associations.
`eks:CreatePodIdentityAssociation` and `eks:DeletePodIdentityAssociation` | Calls needed to create/destroy Associations on AWS EKS.

# Docker hub

We provide some built docker images in Docker hub:

https://hub.docker.com/r/udhos/eks-auto-pod-id-assoc
