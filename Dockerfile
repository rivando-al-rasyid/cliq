FROM golang:1.26.3-alpine3.23

WORKDIR /app

# Combine package installation, caching cleanup, and build steps
RUN apk add --no-cache git tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux go build -trimpath -ldflags="-s -w" -o ./vanwallet ./cmd/main.go

EXPOSE 8080

CMD ["./vanwallet"]