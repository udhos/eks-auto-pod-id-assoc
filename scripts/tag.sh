
version=$(go run ./cmd/eks-auto-pod-id-assoc -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

git tag v${version}
