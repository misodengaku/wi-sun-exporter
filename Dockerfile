FROM golang:1.20 AS builder
WORKDIR /opt
ADD go.mod /opt/
ADD go.sum /opt/
RUN go mod download
ADD udco2s /opt/udco2s
ADD udco2s-exporter.go /opt/
RUN CGO_ENABLED=0 go build

FROM alpine:3
WORKDIR /opt
COPY --from=builder /opt/udco2s-exporter /opt/udco2s-exporter
ENV LISTEN_ADDR=0.0.0.0:9999
ENTRYPOINT /opt/udco2s-exporter
