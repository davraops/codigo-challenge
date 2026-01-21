FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./main.go

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/worker /worker
USER 65532:65532
ENTRYPOINT ["/worker"]
