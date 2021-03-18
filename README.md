
# Installation

Create a cluster using https://kind.sigs.k8s.io/docs/user/quick-start/
```
GO111MODULE="on" go get sigs.k8s.io/kind@v0.10.0
# This step downloads the images. It will take a few minutes depending on the connection
export KUBECONFIG=
$GOPATH/bin/kind --kubeconfig ./config create cluster
# check the status
kubectl cluster-info --context kind-kind
```

# Tips

List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.

# Links

* https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#container-v1-core 
* https://github.com/spotahome/kooper/tree/master/examples - some samples 
* https://github.com/zalando-incubator/kube-ingress-aws-controller
* https://github.com/jcmoraisjr/haproxy-ingress
* https://kind.sigs.k8s.io/docs/user/ingress/ - creat cluster
* https://github.com/slok/kube-code-generator
* https://kind.sigs.k8s.io/docs/user/quick-start/