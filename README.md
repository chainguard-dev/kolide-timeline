# kolide-timeline

Turn Kolide pipeline logs into a timeline - experimental

## Usage

Download logs from Google Cloud Storage for a specific Kolide device:

```
go run ./cmd/copy-from-gs/main.go \
  --bucket chainguard-kolide-logs \
  --prefix kolide/results/incident_response \
  --device-id=183909 \
  --max-age=72h            
```

Turn those logs into a timeline (CSV format):

```
go run ./cmd/kolide-timeline/main.go kolide/
```
