# webapp-cve-consumer

## Overview

The webapp-cve-consumer is a Go application designed to consume Common Vulnerabilities and Exposures (CVE) data from a Kafka topic and store it in a PostgreSQL database. It's part of a larger system for processing and managing CVE information.

## Features

- Consumes CVE data from a Kafka topic
- Stores CVE records in a PostgreSQL database
- Supports versioning of CVE records
- Implements health checks for liveness and readiness probes
- Designed for deployment in a Kubernetes environment

## Prerequisites

- Go 1.x (version used in the project)
- Docker
- Access to a Kafka cluster
- PostgreSQL database
- Kubernetes cluster (for deployment)

## Configuration

The application is configured using environment variables:

- `KAFKA_BROKERS`: Comma-separated list of Kafka broker addresses
- `KAFKA_TOPIC`: Kafka topic to consume CVE data from
- `DB_HOST`: PostgreSQL host
- `DB_PORT`: PostgreSQL port
- `DB_USER`: PostgreSQL username
- `DB_PASSWORD`: PostgreSQL password
- `DB_NAME`: PostgreSQL database name

## Local Development

1. Clone the repository:
   ```
   git clone https://github.com/your-org/webapp-cve-consumer.git
   cd webapp-cve-consumer
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Set up environment variables (use a `.env` file or export them)

4. Run the application:
   ```
   go run main.go
   ```

## Building the Docker Image

To build the Docker image locally:

```
docker build -t webapp-cve-consumer:latest .
```

## Deployment

The application is designed to be deployed in a Kubernetes environment. Refer to the Helm chart in the `helm-webapp-cve-consumer` repository for deployment configurations.

## Health Checks

- Liveness Probe: `/usr/local/bin/liveness-check.sh`
- Readiness Probe: `/usr/local/bin/readiness-check.sh`

These scripts check the application's health and its ability to connect to Kafka and PostgreSQL.

## CI/CD

The repository includes Jenkins pipeline configurations:

- `Jenkinsfile`: For pull request checks (conventional commits, etc.)
- `Jenkinsfile2`: For building and pushing Docker images on the main branch

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

Please ensure your commit messages follow the conventional commits specification.