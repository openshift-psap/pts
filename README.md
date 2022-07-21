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

If you want/need to adjust kubelet config to specify `cpuManagerPolicy`
and your k8s distribution is OpenShift, you can make use one of the
`ocp-kubelet-*.yaml` files supplied in the `./examples` directory.

To run a benchmark, adjust the `./examples/pts-pod-simple.yaml`
for your needs and create the the PTS pod by:
```
kubectl create -f ./examples/pts-pod-simple.yaml
```


## Getting the Phoronix Test Suite results

The PTS results are exposed via the container logs.

```
kubectl logs $pts_pod | sed -ne '/<?xml version="1.0"?>/,$ p' > ${pts_pod}-results.xml
```


## Processing the Phoronix Test Suite results

Once you retrieve the PTS result XML files, you can process the results by the `ptrp`
(Phoronix Test Results Parser) tool.  To get full help on running the tool, run
`./ptrp --help`.

The scoring statistics option `-s` helps users show wins/losses between PTS result XML files.
For example:

```
f1=results/aws/m5/pts-cpu.xml
f2=results/aws/m5a/pts-cpu.xml
f3=results/aws/m6i/pts-cpu.xml

./ptrp \
  -s \
  --suite-name "CPU" \
  "$f1|m5" \
  "$f2|m5a" \
  "$f3|m6i"
```

Another option is detailed comparison between individual test profiles.  For example, to get
`gnuplot` sources comparing three PTS test results files, use:

```
f1=results/aws/m5/pts-cpu.xml
f2=results/aws/m5a/pts-cpu.xml
f3=results/aws/m6i/pts-cpu.xml

./ptrp \
  -o output_dir \
  -f gp \
  "$f1|m5" \
  "$f2|m5a" \
  "$f3|m6i"
```
