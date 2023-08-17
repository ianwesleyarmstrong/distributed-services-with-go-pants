FROM golang:1.21-alpine

WORKDIR /app

COPY src.go.cmd.proglog/bin@environment=linux_go /app/proglog

CMD [ "/app/proglog" ]