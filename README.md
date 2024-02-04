# Simple Retro API

API to Simple Retro website

## Swagger

Install [swaggo](https://github.com/swaggo/swag) if not installed in your system:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

To update Swagger documentation run:
```bash
swag init -g internal/server/server.go
```

To format Swagger comments:
```bash
swag fmt
```

The documentation could be accessed in `/swagger/index.html`
