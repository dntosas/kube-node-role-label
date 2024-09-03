FROM "gcr.io/distroless/static:nonroot"
WORKDIR /
COPY kube-node-role-label .
USER 65532:65532
ENTRYPOINT ["/kube-node-role-label"]

