FROM golang:alpine as builder

RUN mkdir /build 

ADD . /build/
WORKDIR /build 

RUN GOOS=linux GOARCH=arm GOARM=7 go build -o main .

FROM alpine

COPY --from=builder /build/main /app/
COPY --from=builder /build/*.html /app/
WORKDIR /app
CMD ["./main"]