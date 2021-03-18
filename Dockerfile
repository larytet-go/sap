FROM golang:1.14

RUN mkdir /build 
WORKDIR /build
COPY *.go ./
COPY go.mod ./

RUN go mod download
RUN cat go.mod
RUN GOOS=linux CGO_ENABLED=1 GOARCH=amd64 go build -a -o /build ./
