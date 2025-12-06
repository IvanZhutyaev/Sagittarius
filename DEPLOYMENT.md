# Deployment Guide - Стрелец

## Production Deployment

### Prerequisites

- Kubernetes cluster (v1.24+)
- kubectl configured
- Helm 3.x (optional)
- Docker registry access

### Step 1: Build and Push Images

```bash
# Build all service images
docker build -t your-registry/sagittarius-budget:latest -f services/budget/Dockerfile .
docker build -t your-registry/sagittarius-bidder:latest -f services/bidder/Dockerfile .
docker build -t your-registry/sagittarius-auction-engine:latest -f services/auction-engine/Dockerfile .
docker build -t your-registry/sagittarius-notification:latest -f services/notification/Dockerfile .
docker build -t your-registry/sagittarius-api-gateway:latest -f services/api-gateway/Dockerfile .

# Push to registry
docker push your-registry/sagittarius-budget:latest
docker push your-registry/sagittarius-bidder:latest
docker push your-registry/sagittarius-auction-engine:latest
docker push your-registry/sagittarius-notification:latest
docker push your-registry/sagittarius-api-gateway:latest
```

### Step 2: Configure Secrets

```bash
# Create Kubernetes secrets
kubectl create secret generic sagittarius-secrets \
  --from-literal=database-url='postgres://...' \
  --from-literal=jwt-secret='your-secret-key' \
  --from-literal=redis-url='redis://...'
```

### Step 3: Deploy Infrastructure

```bash
# Deploy PostgreSQL (or use managed service)
kubectl apply -f k8s/postgres.yaml

# Deploy Redis (or use managed service)
kubectl apply -f k8s/redis.yaml

# Deploy Kafka (or use managed service)
kubectl apply -f k8s/kafka.yaml
```

### Step 4: Deploy Services

```bash
# Deploy all services
kubectl apply -f k8s/budget-service.yaml
kubectl apply -f k8s/bidder-service.yaml
kubectl apply -f k8s/auction-engine.yaml
kubectl apply -f k8s/notification-service.yaml
kubectl apply -f k8s/api-gateway.yaml
```

### Step 5: Configure Monitoring

```bash
# Deploy Prometheus
kubectl apply -f k8s/prometheus.yaml

# Deploy Grafana
kubectl apply -f k8s/grafana.yaml

# Deploy Jaeger
kubectl apply -f k8s/jaeger.yaml
```

### Step 6: Configure Ingress

```bash
# Deploy Ingress
kubectl apply -f k8s/ingress.yaml
```

## Health Checks

```bash
# Check all pods
kubectl get pods

# Check service health
kubectl exec -it <pod-name> -- curl http://localhost:8080/health

# View logs
kubectl logs -f <pod-name>
```

## Scaling

```bash
# Scale bidder service
kubectl scale deployment bidder-service --replicas=5

# Scale auction engine
kubectl scale deployment auction-engine --replicas=3
```

## Rollback

```bash
# Rollback deployment
kubectl rollout undo deployment/<service-name>

# Check rollout history
kubectl rollout history deployment/<service-name>
```

## Monitoring Production

- Set up alerts in Prometheus
- Configure Grafana dashboards
- Monitor Jaeger traces
- Set up log aggregation (ELK/Loki)

