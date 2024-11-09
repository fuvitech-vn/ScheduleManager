FROM golang:1.21-alpine AS build
RUN apk add --no-cache gcc musl-dev sqlite sqlite-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .

# Builds the application as a staticly linked one, to allow it to run on alpine
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o run .

# Moving the binary to the 'final Image' to make it smaller
FROM alpine
RUN apk add --no-cache curl
WORKDIR /app

COPY --from=build /build/run .
COPY entrypoint.sh /app/entrypoint.sh
COPY templates ./templates

# Set executable permissions for the script
RUN chmod +x /app/entrypoint.sh

# Set the entrypoint
CMD ["/app/entrypoint.sh"]