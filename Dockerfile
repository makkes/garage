# syntax=docker/dockerfile:1.2

# Build the garage binary
FROM golang:1.20 as builder

RUN mkdir /data && chown 65532:65532 /data

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /
COPY garage .
COPY --from=builder --chown=65532:65532 /data ./data
USER 65532:65532

ENTRYPOINT ["/garage"]
