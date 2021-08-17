FROM golang:1.13 as builder
WORKDIR /build
ADD . /build/
RUN GOPROXY="https://goproxy.io" GO111MODULE=on CGO_ENABLED=0 go build -o dra ./src



FROM alpine:3.9.2
RUN mkdir /app
COPY --from=builder /build/dra /app/
ENTRYPOINT ["/app/dra"]