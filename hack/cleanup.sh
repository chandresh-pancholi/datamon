kubectl delete -f hack/k8s/dev.yaml
kubectl delete -f hack/k8s/pv.yaml
kubectl delete -f hack/datamon-volume/datamon-pvc.yaml
kubectl delete -f hack/datamon-volume/datamon-sc.yaml
kubectl delete -f deploy/node.yaml
kubectl delete -f deploy/controller.yaml
