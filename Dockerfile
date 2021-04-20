FROM golang:1.14 AS builder
LABEL stage=builder
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build controller.go


FROM golang:1.14 AS release
ENV USER_UID=1001 USER_NAME=agent
COPY --from=builder /workspace/controller /usr/local/bin/
RUN chmod +x /usr/local/bin/controller
ENTRYPOINT ["/usr/local/bin/controller"]
