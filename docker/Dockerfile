# 1. Build Container
FROM golang:1.22 AS build

# Set build arguments
ARG VERSION
ARG CI_JOB_TOKEN
ARG CI_SERVER_HOST

# Copy the go.mod and go.sum files first to cache the modules
COPY go.mod go.sum /home/src/
WORKDIR /home/src
RUN go mod download

# Copy the rest of your application's source code
COPY . /home/src

# Build the application. Disable CGO and enable static binary output.
RUN CGO_ENABLED=1 go build -v -o dydx-publisher -ldflags="-w -s"

# 2. Runtime Container
FROM alpine as dydx-publisher

# Set environment variables
ENV TZ=Europe/Vilnius \
    PATH="/home/app:${PATH}"

# Install necessary packages
RUN apk add --update --no-cache \
    tzdata \
    ca-certificates \
    bash \
    && \
    cp --remove-destination /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo "${TZ}" > /etc/timezone

# Set working directory
WORKDIR /home/app

# Copy the built application from the build container
COPY --from=build /home/src/dydx-publisher .

# Ensure the application is executable
RUN chmod +x /home/app/dydx-publisher

# Command to run
CMD ["./dydx-publisher", "start"]