# Export asset for local learning only; the academy never builds student Dockerfiles.
FROM golang:1.27.2-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/linkhub .

FROM scratch
COPY --from=build /out/linkhub /linkhub
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65532:65532
EXPOSE 8097
ENTRYPOINT ["/linkhub"]
