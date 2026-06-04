# REST API

`gmd serve` starts an HTTP server on `:8181`.

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Liveness check |
| `/status` | GET | Index and collection health |
| `/search` | POST | Full-text search |
| `/vsearch` | POST | Vector search |
| `/query` | POST | Full hybrid pipeline |
| `/documents/{path}` | GET | Get document by path |
| `/collections` | GET | List collections |
| `/update` | POST | Trigger reindex |
