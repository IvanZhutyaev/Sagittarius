#!/bin/bash

# Create Kafka topics for the system

KAFKA_BROKER=${KAFKA_BROKER:-localhost:9092}

echo "Creating Kafka topics..."

# Auction events
docker exec -it sagittarius-kafka-1 kafka-topics --create \
  --bootstrap-server $KAFKA_BROKER \
  --topic auction-events \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists

# Bid submitted
docker exec -it sagittarius-kafka-1 kafka-topics --create \
  --bootstrap-server $KAFKA_BROKER \
  --topic bid-submitted \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists

# Budget events
docker exec -it sagittarius-kafka-1 kafka-topics --create \
  --bootstrap-server $KAFKA_BROKER \
  --topic budget-events \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists

# Dead Letter Queue
docker exec -it sagittarius-kafka-1 kafka-topics --create \
  --bootstrap-server $KAFKA_BROKER \
  --topic dead-letter-queue \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists

echo "Kafka topics created successfully!"

