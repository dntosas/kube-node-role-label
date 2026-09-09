# The binary is built outside this Dockerfile (goreleaser, or scripts/docker-build.sh)
# and laid out per platform in the build context as <os>/<arch>/kube-node-role-label,
# which is what `docker buildx` exposes through TARGETPLATFORM. This keeps the
# image minimal and lets one build produce a multi-arch manifest.
FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETPLATFORM

LABEL org.opencontainers.image.source="https://github.com/dntosas/kube-node-role-label" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.description="Derive node-role.kubernetes.io/* labels from existing node labels"

COPY ${TARGETPLATFORM}/kube-node-role-label /kube-node-role-label

USER 65532:65532
ENTRYPOINT ["/kube-node-role-label"]
