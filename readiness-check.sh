#!/bin/bash

# Kafka 
if ! kafka-topics.sh --bootstrap-server "${KAFKA_BROKERS}" --list &> /dev/null; then
  echo "Kafka is not reachable"
  exit 1
fi
echo "Kafka is reachable"

# Kafka topic
if ! kafka-topics.sh --bootstrap-server "${KAFKA_BROKERS}" --list | grep -q "${KAFKA_TOPIC}"; then
  echo "Kafka topic ${KAFKA_TOPIC} does not exist"
  exit 1
fi
echo "Kafka topic ${KAFKA_TOPIC} exists"

# Postgres connection check
PGPASSWORD="${DB_PASSWORD}" pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}"
if [ $? -ne 0 ]; then
  echo "PostgreSQL is not ready"
  exit 1
fi
echo "PostgreSQL is ready"

# All checks
exit 0
