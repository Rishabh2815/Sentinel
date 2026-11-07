FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN go mod download
RUN go build -o gateway .

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/gateway .
EXPOSE 8080
CMD ["./gateway"]
