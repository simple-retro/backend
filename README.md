# 👨‍💻 Simple Retro

Backend service for the Simple Retro website. This API provides endpoints for managing retrospectives, including creating and managing retrospective sessions, questions, answers, and real-time updates via WebSocket.

## 🔥 | Running the project

To run Simple Retro API, you need to have [Golang](https://go.dev/) in your machine

1. 🧹 Clone the repository

```bash
git clone git@github.com:simple-retro/backend.git
```

2. 💻 Installing the dependencies

```bash
go get .
```

3. 📝 Set up environment variables

Copy the test environment file to create your local `.env` file:

```bash
cp config/test.env config/.env
```

4. 🏃‍♂️ Running

```bash
go run main.go
```

### 📖 Swagger

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

### 📦 Buiding for production

To deploy the Simple Retro API, use Docker image.

```bash
docker build -t backend:<version> .
```

```bash
docker compose up -d
```

## 🧪 | Running Tests

### Unit Tests

Run unit tests with:

```bash
go test -v $(go list ./... | grep -v /integration_test)
```

### Integration Tests

Integration tests require the service to be running first.

1. Start the service:

```bash
go run main.go
```

2. In another terminal, run the integration tests:

```bash
API_BASE_URL=http://127.0.0.1:8080 go test -v ./integration_test/...
```

### Testing with Act

You can use [act](https://github.com/nektos/act) to test GitHub Actions locally.

For unit tests:

```bash
act -j unit-tests
```

For integration tests, you need to use the `--bind` flag:

```bash
act -j integration-tests --bind
```

## ⚖️ | License

Distributed under the MIT License. See `LICENSE` for more information.

---
