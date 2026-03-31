# APIpost And Swagger Guide

## Generated Swagger Docs

These files are generated from handler annotations by swaggo:

- `docs/swagger/docs.go`
- `docs/swagger/swagger.json`
- `docs/swagger/swagger.yaml`

## APIpost Import

You can import either of these into APIpost:

- local generated file: `docs/swagger/swagger.json`
- local mirrored file: `docs/apipost/js.json`
- running service URL: `http://127.0.0.1:8080/swagger/doc.json`

## Swagger UI

When `configs/config.yaml` sets `swagger.enabled: true`, open:

- `http://127.0.0.1:8080/swagger/index.html`

## Regenerate Docs

```bash
go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g main.go -o docs/swagger --parseInternal
```

## Config Switch

```yaml
swagger:
  enabled: true
  path: /swagger
```

Set `enabled: false` to disable the Swagger routes.

## Default Account

- username: `admin`
- password: `Admin@123`