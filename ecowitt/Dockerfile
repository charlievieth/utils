# syntax=docker/dockerfile:1

FROM golang:1.24
# FROM golang:1.24-alpine

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
COPY *.go ./
COPY pkg ./pkg
RUN go mod download

# Build
RUN GOOS=linux go build -o /bin/ecowitt

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/reference/dockerfile/#expose
EXPOSE 8080

# Run
CMD ["/bin/ecowitt"]
