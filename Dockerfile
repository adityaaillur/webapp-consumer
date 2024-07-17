# Stage 1
FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache git
ENV GOARCH=amd64
ENV GOOS=linux
RUN go mod download
RUN go build -o /app/main .

# Stage 2
FROM wurstmeister/kafka:latest AS kafka-tools

# Stage 3
FROM alpine:latest
RUN apk --no-cache add ca-certificates wget bash openjdk11-jre postgresql-client && \
    wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
    wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.34-r0/glibc-2.34-r0.apk && \
    apk add --force-overwrite glibc-2.34-r0.apk
COPY --from=kafka-tools /opt/kafka_2.13-2.8.1 /opt/kafka
COPY --from=builder /app/main /usr/local/bin/main
COPY readiness-check.sh /usr/local/bin/readiness-check.sh
COPY liveness-check.sh /usr/local/bin/liveness-check.sh
RUN apk add --no-cache dos2unix && \
    dos2unix /usr/local/bin/readiness-check.sh /usr/local/bin/liveness-check.sh && \
    chmod +x /usr/local/bin/main /usr/local/bin/readiness-check.sh /usr/local/bin/liveness-check.sh


ENV KAFKA_HOME=/opt/kafka
ENV PATH=$KAFKA_HOME/bin:$PATH

CMD ["/usr/local/bin/main"]
