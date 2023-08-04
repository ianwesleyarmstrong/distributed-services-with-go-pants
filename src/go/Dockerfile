FROM golang:1.20

WORKDIR /app

COPY src.go.cmd.proglog/bin@environment=linux_docker /app/proglog

CMD [ "/app/proglog" ]