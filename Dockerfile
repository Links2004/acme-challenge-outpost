FROM --platform=$BUILDPLATFORM golang:1.27.3 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/acme-challenge-outpost .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/acme-challenge-outpost /acme-challenge-outpost
EXPOSE 8080 9000
USER nonroot:nonroot
ENTRYPOINT ["/acme-challenge-outpost"]
CMD ["-config", "/etc/acme-challenge-outpost/config.yaml"]
