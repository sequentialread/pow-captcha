
FROM golang:1.16-alpine as build
ARG GOARCH=
ARG GO_BUILD_ARGS=

RUN mkdir /build
WORKDIR /build
RUN apk add --update --no-cache ca-certificates git
COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
RUN  go get && go build -v $GO_BUILD_ARGS -o /build/sequentialread-pow-captcha .

FROM alpine
WORKDIR /app
COPY --from=build /build/sequentialread-pow-captcha /app/sequentialread-pow-captcha
COPY static /app/static
COPY PoW_Captcha_API_Tokens /app/PoW_Captcha_API_Tokens
RUN chmod +x /app/sequentialread-pow-captcha
ENTRYPOINT ["/app/sequentialread-pow-captcha"]