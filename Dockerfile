FROM golang:1.17-alpine

WORKDIR /build
COPY go.* ./
RUN go mod download

COPY . . 
RUN go build -o main .

CMD ["./main"]
