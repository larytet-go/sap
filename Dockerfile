FROM golang:1.15

RUN mkdir /build 
WORKDIR /build

COPY go.mod ./
RUN go mod download
RUN cat go.mod
RUN cat go.sum

COPY *.go ./

RUN GOOS=linux CGO_ENABLED=1 GOARCH=amd64 go build -a -o ./ingress-controller ./
RUN ls -al
# RUN go test -v -failfast ./...

CMD ["./ingress-controller"]
