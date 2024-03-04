FROM golang:alpine as builder

WORKDIR /app/go

# COPY go.mod, go.sum and download the dependencies
COPY go.* ./
RUN go mod download

# COPY All things inside the project and build
COPY . .

RUN go build -o /app/go/build/attendence .
RUN touch /app/go/build/.env && cp -r static templates /app/go/build/

FROM scratch
COPY --from=builder /app/go/build/ /app/

EXPOSE 5000
WORKDIR /app
ENTRYPOINT [ "/app/attendence" ]
