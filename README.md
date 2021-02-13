# bristle

bristle is a grpc service which can process and store arbitrary protobuf messages within Clickhouse.

- loads protobuf descriptors at runtime, linking and validating them with Clickhouse tables
- batches writes to Clickhouse to avoid overwhelming it with merges
- buffers data during instability or outages of Clickhouse


bristle is intended to be a generic Clickhouse data consumer that allows you to safely and efficiently iterate on your full data pipeline, just by using compiled protobuf descriptors.

## Why?

UPLOL built bristle to handle a few main use cases:

- Having a reliable and low-configuration option for experimenting and iterating with data in Clickhouse. We have good experience and usage of protobufs internally, so building our data infrastructure around them is a logical extension.
- Ingesting and processing large amounts of multi-tenant data in a high-availability and high-integrity environment.
- Simple top-level observability ingestion-service for metrics and logs. Clickhouse is a fantastic data-store for observability data, and using bristle we can trivially spin up new metric, log, or event collector services that require very little logic.
