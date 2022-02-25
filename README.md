# Phoronix Test Suite container on CentOS Stream

The Phoronix Test Suite (PTS) container on CentOS Stream provides an
opinionated set of pre-installed tests suites for PTS benchmarking on CentOS
Stream base image.

The resulting test image will vary greatly in size based on the selected
set of suites.


## Building the image

First, choose one or more test suites supplied in
`./assets/var/lib/phoronix-test-suite/test-suites/local`.

For example, to build the container image with `micro` and
`single-threaded-mini` test suites, run:
```
make IMAGE=quay.io/user/pts:single-threaded-mini PTS_TEST_SUITE="local/micro local/single-threaded-mini" image
```


## Running the Phoronix Test Suite

The `./manifests` directory contains manifests necessary for allowing the
provided k8s client to expose the PTS results after the test suite finishes
via a ConfigMap.  Exposing the results via a ConfigMap has its advantages,
but also comes with limitations and the inconvenience to create these
manifests.  Based on feedback and experience, this functionality may be
removed in the future.  The results are also exposed via the container logs,
therefore the manifests are not strictly needed for benchmarking.

If you want/need to adjust kubelet config to specify `cpuManagerPolicy`
and your k8s distribution is OpenShift, you can make use one of the
`ocp-kubelet-*.yaml` files supplied in the `./examples` directory.

To run a benchmark, adjust the `./examples/pts-pod-simple.yaml`
for your needs and create the the PTS pod by:
```
kubectl create -f ./examples/pts-pod-simple.yaml
```


## Getting the Phoronix Test Suite results

### From ConfigMaps

```
kubectl get cm/$pts_results_config_map -o "jsonpath={.data['composite\.xml']}"
```

### From Pod logs

```
kubectl logs $pts_pod | sed -ne '/<?xml version="1.0"?>/,$ p' > ${pts_pod}-results.xml
```
