# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

# Cache module downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Pin templ to the exact version this module was built against
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020
RUN templ generate
RUN CGO_ENABLED=0 go build -trimpath -o /out/app .

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H appuser
WORKDIR /app
COPY --from=build /out/app ./app
USER appuser

ENTRYPOINT ["./app"]
