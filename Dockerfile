FROM golang:1.20

COPY . /app
WORKDIR /app
RUN go build -o kubevirt-apiserver-proxy .

ENTRYPOINT [ "/app/kubevirt-apiserver-proxy"]