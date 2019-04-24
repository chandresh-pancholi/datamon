kubectl create -f hack/k8s/dev.yaml
kubectl create -f hack/k8s/pv.yaml
kubectl create -f hack/datamon-volume/datamon-pvc.yaml
kubectl create -f hack/datamon-volume/datamon-sc.yaml
kubectl create -f deploy/node.yaml
kubectl create -f deploy/controller.yaml
