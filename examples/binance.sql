CREATE TABLE binance.trade_events (
    `eventType` LowCardinality(String),
    `eventTime` DateTime64(3) CODEC(DoubleDelta),
    `symbol` LowCardinality(String),
    `tradeID` UInt64 CODEC(DoubleDelta),
    `price` Decimal(38, 8),
    `quantity` Decimal(38, 8),
    `buyOrderID` UInt64,
    `sellOrderID` UInt64,
    `tradeTime` DateTime64(3) CODEC(DoubleDelta),
    `marketMaker` Nullable(UInt8),
    `M` UInt8
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(eventTime)
ORDER BY (eventTime, symbol, tradeTime)
SETTINGS index_granularity = 8192;