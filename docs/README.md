# Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

    helm repo add eks-auto-pod-id-assoc https://udhos.github.io/eks-auto-pod-id-assoc

Update files from repo:

    helm repo update

Search eks-auto-pod-id-assoc:

    $ helm search repo eks-auto-pod-id-assoc -l --version ">=0.0.0"
    NAME                                                CHART VERSION   APP VERSION	DESCRIPTION
    eks-auto-pod-id-assoc/eks-auto-pod-id-assoc	        0.0.5           0.0.5       Install eks-auto-pod-id-assoc on Kubernetes cluster

To install the charts:

    helm install eks-auto-pod-id-assoc eks-auto-pod-id-assoc/eks-auto-pod-id-assoc
    #            ^                     ^                     ^
    #            |                     |                      \_______ chart
    #            |                     |
    #            |                      \_____________________________ repo
    #            |
    #             \___________________________________________________ release

To uninstall the charts:

    helm uninstall eks-auto-pod-id-assoc

# Source

<https://github.com/udhos/eks-auto-pod-id-assoc>
