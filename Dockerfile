FROM busybox
WORKDIR /app

COPY mp3watcher .
COPY config.json .

CMD ["/app/mp3watcher"]