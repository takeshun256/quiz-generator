FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o quiz-generator .

FROM alpine:3.21

WORKDIR /app
COPY --from=builder /app/quiz-generator .
COPY templates/ templates/
COPY static/ static/

EXPOSE 8080
CMD ["./quiz-generator"]
