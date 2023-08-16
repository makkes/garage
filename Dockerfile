# syntax=docker/dockerfile:1.2

FROM --platform=$BUILDPLATFORM tonistiigi/xx AS xx

FROM --platform=$BUILDPLATFORM golang:1.20 as builder
ARG TARGETPLATFORM

COPY --from=xx / /

WORKDIR /workspace

COPY cmd/ cmd/
COPY pkg/ pkg/
COPY go.mod go.mod
COPY go.sum go.sum

RUN mkdir /data && chown 65532:65532 /data

ENV CGO_ENABLED=0
RUN xx-go build -o /garage ./cmd/garage/main.go

# Ensure that the binary was cross-compiled correctly to the target platform.
RUN xx-verify --static /garage

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static-debian11:nonroot

WORKDIR /
COPY --from=builder /garage .
COPY --from=builder --chown=65532:65532 /data ./data
USER 65532:65532

ENTRYPOINT ["/garage"]
