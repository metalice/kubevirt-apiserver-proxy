FROM golang:1.20

COPY . /app
WORKDIR /app
RUN go build -o kubevirt-performance-pod .

ENTRYPOINT [ "/app/kubevirt-performance-pod"]