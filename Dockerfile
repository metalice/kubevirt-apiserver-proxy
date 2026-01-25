FROM golang:1.26rc2

COPY . /app
WORKDIR /app
ENV GIN_MODE=release
RUN go build -o kubevirt-apiserver-proxy .

ENTRYPOINT ["/app/kubevirt-apiserver-proxy"]