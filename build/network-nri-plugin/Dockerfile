ARG BASE_IMAGE=registry.nordix.org/cloud-native/network-dra/base-image:latest
ARG BUILD_VERSION=0.0.0-unknown

FROM golang:1.22 as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags -static" -o network-nri-plugin ./cmd/network-nri-plugin

FROM ${BASE_IMAGE}

COPY --from=build /app/network-nri-plugin .

CMD ["./network-nri-plugin", "run"]
