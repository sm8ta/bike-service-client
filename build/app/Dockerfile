FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o bike-service-app ./cmd/main.go

EXPOSE 8081
EXPOSE 50052

CMD ["./bike-service-app"]