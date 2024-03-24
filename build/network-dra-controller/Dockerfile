ARG BASE_IMAGE=registry.nordix.org/cloud-native/network-dra/base-image:latest
ARG BUILD_VERSION=0.0.0-unknown

FROM golang:1.22 as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags -static" -o network-dra-controller ./cmd/network-dra-controller

FROM ${BASE_IMAGE}

COPY --from=build /app/network-dra-controller .

CMD ["./network-dra-controller", "run"]
