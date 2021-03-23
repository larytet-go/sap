# Deployment

Create a cluster using https://kind.sigs.k8s.io/docs/user/quick-start/
Pay attention to the configuration `kind.yaml` - map external ports
```sh
GO111MODULE="on" go get sigs.k8s.io/kind@v0.10.0
# Make sure that no other configurations are impacted
export KUBECONFIG=./config
# $GOPATH/bin/kind --kubeconfig ./config delete cluster
# This step downloads the images. It will take a few minutes depending on the connection
$GOPATH/bin/kind --kubeconfig ./config --config kind.yaml create cluster
```

Check clusters status
```
$GOPATH/bin/kind get clusters
kubectl cluster-info --context kind-kind
```

Build the controller, load the image into the cluster
```sh
docker build -t ingress-controller:mylatest .
$GOPATH/bin/kind load docker-image ingress-controller:mylatest
```

# Usage

Modify environment variable RULES in the ingress-controller.yaml. The variable contains comma separated tuples [ (hostname:service),(hostname:service), ... ]. Intentionaly there is no port number: ingress-controller relies on the containes ports in the pods configurations. ingress-controller ignores whitespaces. There is not rule enforcement. If an HTTP request does not match any rule the controller attempts to match any service with open ports in the cluster.

In the example below rules map `echo` to the service `default/echo-app`
```yaml
env:
- name: "RULES"
    value: " echo : default/echo-app "
```

Start the service 
```sh
kubectl apply -f ./echo.yaml
kubectl apply -f ./ingress-controller.yaml
# kubectl get all
kubectl get pods
```

Try `echo` service
```sh
# Use mapping from the RULES
curl http://127.0.0.1:8080/echo
curl -H "Host: echo" http://127.0.0.1:8080

# Explicit service name
curl http://127.0.0.1:8080/default/echo-app

# If the DNS is configured to resolve host `echo` you can do
curl -H "Host: echo" http://echo:8080
```

# Development

Get the service status 
```sh
curl http://127.0.0.1:8080/ingress
```

Log
```sh
export KUBECONFIG=./config && $GOPATH/bin/stern_linux_amd64 ".*"
```

Restart the controller 
```sh
# kubectl delete pod/ingress-controller && kubectl apply -f ./ingress-controller.yaml && kubectl get all
kubectl scale --replicas=0 deployment.apps/ingress-controller
kubectl scale --replicas=1 deployment.apps/ingress-controller
```

Build and load the ingress container
```sh
docker build -t ingress-controller:mylatest . && \
$GOPATH/bin/kind load docker-image ingress-controller:mylatest && \
kubectl scale --replicas=0 deployment.apps/ingress-controller && \
kubectl apply -f ./ingress-controller.yaml && \
kubectl scale --replicas=1 deployment.apps/ingress-controller  && \
kubectl get all
```

Restart the `echo` service
```sh
kubectl delete pod echo-app && kubectl apply -f ./echo.yaml && kubectl get all
```

```sh
kubectl get pods | grep ingress | awk '{print $1}' | xargs -I{} kubectl exec {}  -- curl --silent http://10.244.0.7:5688
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
* https://stackoverflow.com/questions/11738029/how-do-i-unregister-a-handler-in-net-http/11851973  - custom mux
* https://stackoverflow.com/questions/21978883/how-to-define-custom-mux-with-golang - another example of mux