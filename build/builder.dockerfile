FROM golang:1.22

WORKDIR /app

ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64

COPY build/build_plugins.sh .
RUN chmod +x ./build_plugins.sh

CMD ["./build_plugins.sh"]