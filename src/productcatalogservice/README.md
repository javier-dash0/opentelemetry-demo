# Product Catalog Service

## Fault injection

`GetProduct` can fail lookups on purpose so the demo has a reproducible error to
investigate. This is **opt-in** and controlled by two settings:

| Setting | Where | Effect |
| :--- | :--- | :--- |
| `PRODUCT_CATALOG_FAILURE_PRODUCT_ID` | env var on this service | The product ID that may fail. Unset or empty disables fault injection completely. |
| `productCatalogFailure` | `featureflagservice` | Probability (`0`–`1`) that a lookup of that product ID fails with gRPC `INTERNAL`. |

Both have to be set for a request to fail. Enable the scenario with:

```sh
PRODUCT_CATALOG_FAILURE_PRODUCT_ID=OLJCESPC7Z
```

and set `productCatalogFailure` to the desired probability in the
featureflagservice UI. Leaving the env var empty keeps the flag inert, so a
probability left behind in the featureflagservice database cannot quietly raise
this service's error rate.

When this service is run the output should be similar to the following

```json
{"message":"successfully parsed product catalog json","severity":"info","timestamp":"2022-06-02T23:54:10.191283363Z"}
{"message":"starting grpc server at :3550","severity":"info","timestamp":"2022-06-02T23:54:10.191849078Z"}
```

## Local Build

To build the service binary, run:

```sh
go build -o /go/bin/productcatalogservice/
```

## Docker Build

From the root directory, run:

```sh
docker compose build productcatalogservice
```

## Regenerate protos

> [!NOTE]
> [`protoc`](https://grpc.io/docs/protoc-installation/) is required.

To regenerate gRPC code run:

```sh
go generate
```

## Bump dependencies

To bump all dependencies run:

```sh
go get -u -t ./...
go mod tidy
```
