FROM golang:1.23 as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags -static" -o network-nri-plugin ./cmd/network-nri-plugin

FROM alpine:3.20

COPY --from=build /app/network-nri-plugin .

CMD ["./network-nri-plugin", "run"]
