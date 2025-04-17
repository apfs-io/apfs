# APFS - Autoprocessing File System

[![Build Status](https://github.com/apfs-io/apfs/workflows/Tests/badge.svg)](https://github.com/apfs-io/apfs/actions?workflow=Tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/apfs-io/apfs)](https://goreportcard.com/report/github.com/apfs-io/apfs)
[![GoDoc](https://godoc.org/github.com/apfs-io/apfs?status.svg)](https://godoc.org/github.com/apfs-io/apfs)
[![Coverage Status](https://coveralls.io/repos/github/apfs-io/apfs/badge.svg)](https://coveralls.io/github/apfs-io/apfs)

`apfs-io/apfs` is an advanced, automated file-processing tool designed for high-performance handling of filesystem objects. The project provides a gRPC and RESTful API for managing, processing, and streaming file objects.

## Features

- **Efficient File Processing:** Upload, stream, and process file data seamlessly.
- **Manifest Management:** Manage data manifests by group for organized processing.
- **Object Operations:** Supports retrieving, updating, and deleting filesystem objects.
- **Protocol Buffers:** API defined with `proto3` for cross-language compatibility.
- **REST Integration:** REST endpoints auto-generated via gRPC Gateway.
- **Comprehensive API Documentation:** OpenAPI specs available for easy integration.

## API Overview

### Services

1. **Object Operations**
   - `Head`: Retrieve object metadata.
   - `Get`: Fetch object data and metadata.
   - `Refresh`: Reprocess and refresh object data.
   - `Delete`: Delete objects or their subitems.

2. **Manifest Operations**
   - `SetManifest`: Define or update group manifests.
   - `GetManifest`: Fetch group manifest data.

3. **Data Upload**
   - `Upload`: Stream new file data into the system.

### Protocol Buffers Structure

#### Key Enumerations

- `ResponseStatusCode`:
  - `RESPONSE_STATUS_CODE_OK`: Success.
  - `RESPONSE_STATUS_CODE_FAILED`: Error occurred.
  - `RESPONSE_STATUS_CODE_NOT_FOUND`: Resource not found.

#### Key Messages

- `Data`: Represents file content or metadata.
- `Manifest`: Represents a processing manifest.
- `ObjectID`: Identifies objects within the filesystem.

## API Endpoints

| Method   | Endpoint                   | Description                        |
|----------|----------------------------|------------------------------------|
| `GET`    | `/v1/head/{id}`            | Retrieve object metadata.          |
| `GET`    | `/v1/object/{id}`          | Retrieve object and data.          |
| `PUT`    | `/v1/refresh/{id}`         | Reprocess an object.               |
| `PUT`    | `/v1/manifest/{group}`     | Set a manifest for a group.        |
| `GET`    | `/v1/manifest/{group}`     | Retrieve a group manifest.         |
| `POST`   | `/v1/object`               | Upload new file data.              |
| `DELETE` | `/v1/object/{id}`          | Delete an object or subitems.      |

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
