PROGRAM:=ptrp
SRC:=$(shell find ./cmd -name \*.go)
ARCH:=amd64	# arm64
OUT_DIR:=_output
GOFLAGS:=

DOCKERFILE:=Dockerfile.cs8
REGISTRY:=quay.io
ORG:=openshift-psap
TAG:=micro
IMAGE:=$(REGISTRY)/$(ORG)/pts:$(TAG)
AUTHFILE:=$(HOME)/.docker/config-quay.json
NOCACHE:=--no-cache
# Multiple test suites are separated by spaces accepted (e.g. "local/micro local/single-threaded-mini").
PTS_TEST_SUITE:=local/micro

build: $(PROGRAM)

$(PROGRAM): $(SRC)
	GOFLAGS=$(GOFLAGS) go build -o $(OUT_DIR)/$(PROGRAM) $^

static: $(SRC)
	GOFLAGS=$(GOFLAGS) CGO_ENABLED=0 go build -o $(OUT_DIR)/$(PROGRAM) -a -installsuffix cgo -ldflags '-s' $^

fmt format: $(SRC)
	go fmt $^

vet: $(SRC)
	go vet $^

strip:
	strip $(PROGRAM)

clean:
	go clean
	rm -rf $(OUT_DIR)

image: $(DOCKERFILE)
	podman build $(NOCACHE) --arch $(ARCH) --build-arg=PTS_TEST_SUITE="$(PTS_TEST_SUITE)" -f $(DOCKERFILE) -t $(IMAGE) .

image-push push: 
	podman push --authfile $(AUTHFILE) $(IMAGE)
