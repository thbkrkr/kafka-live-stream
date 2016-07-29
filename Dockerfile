FROM alpine:3.4

COPY kafka-live-stream /app/kafka-live-stream
COPY _static /app/_static

WORKDIR "/app"
ENTRYPOINT ["/app/kafka-live-stream"]