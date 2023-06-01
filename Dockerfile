FROM golang:1.19

COPY . /app
WORKDIR /app
RUN go build -o kubevirt-performance-pod .

ENTRYPOINT [ "/app/kubevirt-performance-pod"]