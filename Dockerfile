FROM golang:1.23-rc-alpine
WORKDIR /app
RUN apk update && \
    apk add --no-cache make=4.4.1-r2 && \
    addgroup --system gouser && \
    adduser --system gouser --ingroup gouser
COPY go.mod go.sum ./
RUN go mod download
COPY . /app
RUN make
EXPOSE 3000
USER gouser:gouser
CMD [ "make", "run" ]
