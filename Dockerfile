FROM golang:1.20

COPY . /app
WORKDIR /app
RUN go build -o kubevirt-proxy-pod .

ENTRYPOINT [ "/app/kubevirt-proxy-pod"]