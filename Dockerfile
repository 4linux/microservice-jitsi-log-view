FROM golang:1.14-alpine
WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
RUN go build -v ./...

FROM alpine:3.12.0 
COPY --from=0 /go/src/app/microservice-jitsi-log-view /usr/bin/
RUN apk --no-cache add tzdata
EXPOSE 8080
CMD ["microservice-jitsi-log-view"]
