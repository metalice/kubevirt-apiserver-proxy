FROM golang:1.24.0

COPY . /app
WORKDIR /app
ENV GIN_MODE=release
RUN go build -o kubevirt-apiserver-proxy .

ENTRYPOINT ["/app/kubevirt-apiserver-proxy"]