# Proxy

## Info

Proxy listen on URL: `http://${address}/proxy`

Flink's `module.yaml` example:

```yaml
kind: io.statefun.endpoints.v2/http
spec:
  functions: proxy/*
  urlPathTemplate: http://${address}/proxy
  maxNumBatchRequests: 1
  transport:
    type: io.statefun.transports.v1/async
    connect: 5s
    call: 600s
    payload_max_bytes: 52428800

```

## Run

### Environment variables

|Variable|Default|Description|
|:------:|:-----:|:---------:|
|STATEFUN_PROXY_DEBUG|""|Debug log level|
