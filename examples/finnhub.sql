CREATE TABLE finnhub.trades (
    `symbol` LowCardinality(String),
    `price` Float64,
    `tradeTime` DateTime64(3) CODEC(DoubleDelta),
    `volume` Float64,
    `tradeConditions` Array(String) DEFAULT []
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(tradeTime)
ORDER BY (tradeTime, symbol)
SETTINGS index_granularity = 8192;