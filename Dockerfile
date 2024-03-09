FROM golang:alpine as builder

WORKDIR /app/go

# COPY go.mod, go.sum and download the dependencies
COPY go.* ./
RUN go mod download
RUN apk add --no-cache ca-certificates

# COPY All things inside the project and build
COPY . .

RUN go build -o /app/go/build/attendence .
RUN touch /app/go/build/.env && cp -r static templates /app/go/build/

FROM scratch 

COPY --from=builder /app/go/build/ /app/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/


WORKDIR /app
ENTRYPOINT [ "/app/attendence" ]
