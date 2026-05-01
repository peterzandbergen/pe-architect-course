FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /teams-operator .

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /teams-operator /teams-operator

USER 1001:1001

ENTRYPOINT ["/teams-operator"]