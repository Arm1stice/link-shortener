FROM golang:alpine AS build
RUN apk add --no-cache git
RUN go get github.com/golang/dep/cmd/dep
RUN go get -u github.com/gobuffalo/packr/...

COPY Gopkg.lock Gopkg.toml /go/src/github.com/wcalandro/link-shortener/
WORKDIR /go/src/github.com/wcalandro/link-shortener/
RUN dep ensure -vendor-only

COPY . /go/src/github.com/wcalandro/link-shortener/

RUN packr
RUN go build -o /bin/link-shortener

FROM alpine
COPY --from=build /bin/link-shortener /bin/link-shortener
ENTRYPOINT "/bin/link-shortener" 
EXPOSE 5000