# kolide-timeline

[![stable](http://badges.github.io/stability-badges/dist/stable.svg)](http://github.com/badges/stability-badges)

kolide-timeline generates a timeline in CSV format from Kolide pipeline logs, using both query timestamps and any
timestamps returned by the queries.

This tool is geared toward security investigations and incident response.

![screenshot](images/sheet.png?raw=true "screenshot")

## Requirements

* Go v1.20 or newer

## Installation

```shell
go install github.com/chainguard-dev/kolide-timeline/cmd/kolide-timeline@latest
go install github.com/chainguard-dev/kolide-timeline/cmd/copy-from-gs@latest
```

## Usage

Timeline generation assumes that pipeline logs have been locally downloaded:

```
kolide-timeline </path/to/device/logs>
```

If your Kolide pipeline logs are stored in Google Cloud Storage, there is a tool to simplify downloading recent logs for a single device:

```
copy-from-gs \
  --bucket chainguard-kolide-logs \
  --prefix kolide/results \
  --device-id=183909 \
  --max-age=72h            
```

To find the device ID, visit https://k2.kolide.com/, click on the Device, and view its URL: it will end in `/inventory/devices/<device id>/overview`. 