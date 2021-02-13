CREATE TABLE IF NOT EXISTS example_table (
    name String,
    type Enum('empty' = 0, 'small' = 1, 'big' = 2),
    timestamp DateTime,
    value Nullable(Int64),
    tags Nested(key String, value String),
    labels Array(String) DEFAULT []
)
ENGINE = MergeTree
PARTITION BY (toStartOfDay(timestamp))
ORDER BY (timestamp, type, name);