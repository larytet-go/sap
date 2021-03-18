FROM golang:1.14

RUN mkdir /build 
WORKDIR /build
COPY *.go ./

