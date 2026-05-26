# Build the manager binary
FROM quay.io/projectquay/golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG APP_BUILD_ROOT

## strictfipsruntime does not work with disabling CGO, which is a best practice in this case
# ENV GOEXPERIMENT=strictfipsruntime
ENV APP_ROOT=$APP_BUILD_ROOT
ENV GOPATH=$APP_ROOT

WORKDIR $APP_ROOT/src/
COPY go.mod ./
COPY go.sum ./
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/
COPY pkg/ pkg/
RUN go mod download && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o ${APP_ROOT}/manager cmd/main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:5b74fce9d6e629942a0c6dc0f546c193e70d7f974d999a48c948c53dd3d36362 AS deploy

ARG VERSION
ARG APP_BUILD_ROOT

WORKDIR /
COPY --from=builder ${APP_BUILD_ROOT}/manager .
COPY LICENSE /licenses/license.txt
USER 65532:65532

ENTRYPOINT ["/manager"]
