FROM golang:1.20.3-alpine as build

ENV GOFLAGS="-mod=vendor"
ENV CGO_ENABLED=0

ADD . /build
WORKDIR /build

RUN apk add --no-cache --update ca-certificates
RUN cd app && go build -o /build/google-maps-to-waze

FROM scratch
COPY --from=build /build/google-maps-to-waze /srv/google-maps-to-waze
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /srv
ENTRYPOINT ["/srv/google-maps-to-waze"]