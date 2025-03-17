FROM golang:1.21.10

COPY . /app
WORKDIR /app
ENV GIN_MODE=release
RUN go build -o kubevirt-apiserver-proxy .

ENTRYPOINT ["/app/kubevirt-apiserver-proxy"]