# kolide-timeline

[![stable](http://badges.github.io/stability-badges/dist/stable.svg)](http://github.com/badges/stability-badges)

Use Kolide pipeline logs as a timeline source for incident response. This includes query timestamps as well as any timestamps returned by queries.

## Requirements

* Go v1.20 or newer

## Installation

```shell
go install github.com/chainguard-dev/kolide-timeline/cmd/kolide-timeline@latest
go install github.com/chainguard-dev/kolide-timeline/cmd/copy-from-gs@latest
```

## Usage

kolide-timeline operates on locally download files:

```
kolide-timeline </path/to/device/logs>
```

If your Kolide pipeline logs are stored in Google Cloud Storage, there is a tool to simplify downloading logs for a single device:

```
copy-from-gs \
  --bucket chainguard-kolide-logs \
  --prefix kolide/results \
  --device-id=183909 \
  --max-age=72h            
```

