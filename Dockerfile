FROM golang:1.22-alpine as builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main cmd/server/main.go

FROM alpine

COPY --from=builder /app/main /app/main
CMD ["/app/main"]