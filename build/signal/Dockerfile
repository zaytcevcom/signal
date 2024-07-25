FROM golang:1.22 as build

ENV BIN_FILE /opt/app/bin
ENV CODE_DIR /go/src/

WORKDIR ${CODE_DIR}

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY .. ${CODE_DIR}

ARG LDFLAGS
RUN CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o ${BIN_FILE} cmd/signal/*

FROM alpine:latest

LABEL SERVICE="signal"
LABEL MAINTAINERS="zaytcev.com"

ENV BIN_FILE /opt/app/bin
COPY --from=build ${BIN_FILE} ${BIN_FILE}

ENV CONFIG_FILE /etc/app/config.json
COPY ./configs/signal/config.json ${CONFIG_FILE}

EXPOSE 1989

CMD ${BIN_FILE} -config ${CONFIG_FILE}
