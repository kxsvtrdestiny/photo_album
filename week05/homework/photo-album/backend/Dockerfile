FROM golang:1.25-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/*.go ./
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -o /out/photo-album .

FROM alpine:3.23
COPY --from=build /out/photo-album /usr/local/bin/photo-album
COPY backend/docker-entrypoint.sh /usr/local/bin/photo-album-entrypoint
RUN sed -i 's/\r$//' /usr/local/bin/photo-album-entrypoint
ENTRYPOINT ["sh", "/usr/local/bin/photo-album-entrypoint"]
