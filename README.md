# Exodus Rush - Character Service

Go-based microservice for managing character state, movement, and crossing logic in the Exodus Rush game.

## Overview

The Character Service is responsible for:
- Managing character positions in the game world
- Handling character movement requests
- Validating crossing attempts by checking sea state
- Storing character data (in-memory or PostgreSQL)

## API Endpoints

### Health Check
```
GET /health
```
Returns service health status.

### Move Character
```
POST /move
Content-Type: application/json

{
  "character_id": "char-123",
  "x": 100.5,
  "y": 200.3
}
```
Moves a character to a new position.

### Get Character Position
```
GET /position/:characterId
```
Returns the current position of a character.

### Initiate Crossing
```
POST /cross
Content-Type: application/json

{
  "character_id": "char-123"
}
```
Attempts to cross the Red Sea. Checks the sea-state-service to verify if the sea is split.

### Get Character Status
```
GET /status/:characterId
```
Returns the full status of a character including position, state, and crossing ability.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Service port | `8081` |
| `DB_HOST` | PostgreSQL host | - |
| `DB_USER` | Database username | - |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | `exodus_characters` |

If database credentials are not provided, the service falls back to in-memory storage.

## Dependencies

### External Services
- **sea-state-service** (`:8080`) - Used to check if the sea is split before allowing crossing
- **postgres-db** (`:5432`) - Optional persistent storage for character data

### Go Packages
- `github.com/gorilla/mux` - HTTP router
- `github.com/lib/pq` - PostgreSQL driver

## Development

### Prerequisites
- Go 1.21+
- Docker (for containerization)
- Access to sea-state-service

### Build Locally
```bash
go mod download
go build -o character-service
./character-service
```

### Build Docker Image
```bash
docker build -t stealthymcstelath/exodus-rush-character-service:latest .
```

### Run with Docker
```bash
docker run -p 8081:8081 \
  -e DB_HOST=postgres \
  -e DB_USER=exodus \
  -e DB_PASSWORD=secret \
  stealthymcstelath/exodus-rush-character-service:latest
```

## Kubernetes Deployment

### Deploy to Cluster
```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### Check Status
```bash
kubectl get pods -n passover -l app=character-service
kubectl get svc -n passover character-service
```

### View Logs
```bash
kubectl logs -n passover -l app=character-service --tail=100
```

## Architecture

The service runs with 3 replicas for high availability. It maintains character state either in:
- **PostgreSQL** (when configured) - Shared state across replicas
- **In-memory** (fallback) - Per-pod state

### Request Flow: Crossing Attempt
1. Client sends `POST /cross` with character ID
2. Service retrieves character from store
3. Service calls `sea-state-service:8080/status` to check sea state
4. If sea is "split", character can cross
5. Character state updated to "crossing" or "waiting"
6. Response sent back to client

## Testing

### Health Check
```bash
curl http://localhost:8081/health
```

### Create and Move Character
```bash
curl -X POST http://localhost:8081/move \
  -H "Content-Type: application/json" \
  -d '{"character_id": "moses", "x": 0, "y": 0}'
```

### Attempt Crossing
```bash
curl -X POST http://localhost:8081/cross \
  -H "Content-Type: application/json" \
  -d '{"character_id": "moses"}'
```

### Get Character Status
```bash
curl http://localhost:8081/status/moses
```

## Design Notes

### Stateless Design
The character service is designed to be stateless when using PostgreSQL as the backend. Multiple replicas can serve requests without state synchronization issues.

### Crossing Logic
The service delegates sea state verification to the sea-state-service. This separation of concerns allows:
- Sea state logic to be managed independently
- Character service to remain focused on character management
- Clear service boundaries

### Error Handling
- Falls back to in-memory storage if database is unavailable
- Returns appropriate HTTP status codes
- Logs errors for debugging

## Production Considerations

1. **Database**: Always configure PostgreSQL in production for shared state
2. **Monitoring**: Health endpoints enable liveness/readiness probes
3. **Scaling**: Can scale horizontally with 3+ replicas
4. **Dependencies**: Ensure sea-state-service is reachable

## License

Part of the Exodus Rush CTF Challenge - Bonez Platform
