# 👨‍💻 Simple Retro

## 🔥 | Running the project

To run Simple Retro API, you need to have [Golang](https://go.dev/) in your machine. We recommend at least version 1.21.

1. 🧹 Clone the repository

```bash
git clone git@github.com:simple-retro/backend.git
```

2. 💻 Installing the dependencies

```bash
go get .
```

3. 🏃‍♂️ Running

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
docker build -t simple-retro-api .
```

```bash
docker compose up -d
```

## 🔨 | Made With 
 
- [Go](https://go.dev/) 
- [Gin](https://github.com/gin-gonic/gin) 
- [Swag](https://github.com/swaggo/swag)
- [SQLite3](https://github.com/mattn/go-sqlite3)
- [Gorilla Websocket](https://github.com/gorilla/websocket)

## ⚖️ | License

Distributed under the MIT License. See `LICENSE` for more information.

---
