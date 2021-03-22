
# Installation

Create a cluster using https://kind.sigs.k8s.io/docs/user/quick-start/
```
GO111MODULE="on" go get sigs.k8s.io/kind@v0.10.0
# Make sure that no other configurations are impacted
export KUBECONFIG=./config
# This step downloads the images. It will take a few minutes depending on the connection
$GOPATH/bin/kind --kubeconfig ./config delete cluster
$GOPATH/bin/kind --kubeconfig ./config --config kind.yaml create cluster

# list clusters 
$GOPATH/bin/kind get clusters
# check the status
kubectl cluster-info --context kind-kind

# Build the image 
docker build -t ingress-controller:mylatest .
# Load the image into the cluster
$GOPATH/bin/kind load docker-image ingress-controller:mylatest


# Start the service 
kubectl apply -f ./echo.yaml && kubectl get all
kubectl apply -f ./ingress-controller.yaml && kubectl get all
# kubectl get all
kubectl get pods

# Log
export KUBECONFIG=./config && $GOPATH/bin/stern_linux_amd64 ingress


# Restart the service
# kubectl delete pod/ingress-controller && kubectl apply -f ./ingress-controller.yaml && kubectl get all
kubectl scale --replicas=0 deployment.apps/ingress-controller
kubectl scale --replicas=1 deployment.apps/ingress-controller
```

# Tips

List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.

```
docker build -t ingress-controller:mylatest . && \
$GOPATH/bin/kind load docker-image ingress-controller:mylatest && \
kubectl scale --replicas=0 deployment.apps/ingress-controller && \
kubectl scale --replicas=1 deployment.apps/ingress-controller  && \
kubectl get all
```

```
kubectl get pods | grep ingress | awk '{print $1}' | xargs -I{} kubectl exec {}  -- curl http://echo-app:5688
```

# Links

* https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#container-v1-core 
* https://github.com/spotahome/kooper/tree/master/examples - some samples 
* https://github.com/zalando-incubator/kube-ingress-aws-controller
* https://github.com/jcmoraisjr/haproxy-ingress
* https://kind.sigs.k8s.io/docs/user/ingress/ - creat cluster
* https://github.com/slok/kube-code-generator
* https://kind.sigs.k8s.io/docs/user/quick-start/
* https://github.com/kubernetes/client-go/issues/741  - package dependencies k8s 1.19
* https://app.slack.com/client/T09NY5SBT/CEKK1KTN2  - kind slack
* https://github.com/kubernetes/sample-controller - another example
* https://itnext.io/building-an-operator-for-kubernetes-with-the-sample-controller-b4204be9ad56 - Roles example
* https://github.com/kubernetes/api/blob/fd88418e43d2da5bce86eeeae341d6477c63e07a/core/v1/types.go  - k8s API