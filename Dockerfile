FROM golang:1.15-buster as builder

ENV GO111MODULE on
ENV GOSUMDB off
COPY . /home/keycloak
WORKDIR /home/keycloak
RUN go build -o keycloak-lark-adapter cmd/cmd.go

FROM debian:buster-slim
RUN apt-get update && apt-get install -y openssl ca-certificates
WORKDIR /home/keycloak
COPY --from=builder /home/keycloak/keycloak-lark-adapter /home/keycloak/
CMD ["/home/keycloak/keycloak-lark-adapter"]
