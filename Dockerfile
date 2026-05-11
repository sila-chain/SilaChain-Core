# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# Build Sila in a stock Go builder container
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev linux-headers git

# Get dependencies - will also be cached if we won't change go.mod/go.sum
COPY go.mod /sila/
COPY go.sum /sila/
RUN cd /sila && go mod download

ADD . /sila
RUN cd /sila && go run build/ci.go install -static ./cmd/sila

# Pull Sila into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /sila/build/bin/sila /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["sila"]

# Add some metadata labels to help programmatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
