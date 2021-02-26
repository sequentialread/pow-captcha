
FROM golang:1.15.2-alpine as build
ARG GOARCH=
ARG GO_BUILD_ARGS=

RUN mkdir /build
WORKDIR /build
RUN apk add --update --no-cache ca-certificates git \
  && go get golang.org/x/crypto/scrypt
COPY . .
RUN  go build -v $GO_BUILD_ARGS -o /build/sequentialread-pow-captcha .

FROM alpine
WORKDIR /app
COPY --from=build /build/sequentialread-pow-captcha /app/sequentialread-pow-captcha
COPY --from=build /build/static /app/static
COPY PoW_Captcha_API_Tokens /app/PoW_Captcha_API_Tokens
RUN chmod +x /app/sequentialread-pow-captcha
ENTRYPOINT ["/app/sequentialread-pow-captcha"]