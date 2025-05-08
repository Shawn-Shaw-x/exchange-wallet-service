DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'uint256') THEN
            CREATE DOMAIN UINT256 AS NUMERIC
                CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
        ELSE
            ALTER DOMAIN UINT256 DROP CONSTRAINT uint256_check;
            ALTER DOMAIN UINT256 ADD
                CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
        END IF;
    END
$$;

CREATE TABLE IF NOT EXISTS business
(
    guid         VARCHAR PRIMARY KEY,
    business_uid VARCHAR NOT NULL,
    notify_url   VARCHAR NOT NULL,
    timestamp    INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS tokens_timestamp ON business (timestamp);
CREATE UNIQUE INDEX IF NOT EXISTS business_uid ON business (business_uid);

CREATE TABLE IF NOT EXISTS blocks
(
    hash        VARCHAR PRIMARY KEY,
    parent_hash VARCHAR NOT NULL UNIQUE,
    number      UINT256 NOT NULL UNIQUE CHECK (number > 0),
    timestamp   INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS blocks_number ON blocks (number);
CREATE INDEX IF NOT EXISTS blocks_timestamp ON blocks (timestamp);

CREATE TABLE IF NOT EXISTS reorg_blocks
(
    hash        VARCHAR PRIMARY KEY,
    parent_hash VARCHAR NOT NULL UNIQUE,
    number      UINT256 NOT NULL UNIQUE CHECK (number > 0),
    timestamp   INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS reorg_blocks_number ON reorg_blocks (number);
CREATE INDEX IF NOT EXISTS reorg_blocks_timestamp ON reorg_blocks (timestamp);

CREATE TABLE IF NOT EXISTS transactions
(
    guid          VARCHAR PRIMARY KEY,
    block_hash    VARCHAR NOT NULL,
    block_number  UINT256 NOT NULL CHECK (block_number > 0),
    hash          VARCHAR NOT NULL,
    from_address  VARCHAR NOT NULL,
    to_address    VARCHAR NOT NULL,
    token_address VARCHAR NOT NULL,
    token_id      VARCHAR NOT NULL,
    token_meta    VARCHAR NOT NULL,
    fee           UINT256 NOT NULL,
    amount        UINT256 NOT NULL,
    status        VARCHAR NOT NULL,
    tx_type       VARCHAR NOT NULL,
    timestamp     INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS transactions_hash ON transactions (hash);
CREATE INDEX IF NOT EXISTS transactions_timestamp ON transactions (timestamp);

CREATE TABLE IF NOT EXISTS addresses
(
    guid         VARCHAR PRIMARY KEY,
    address      VARCHAR UNIQUE NOT NULL,
    address_type VARCHAR(10)    NOT NULL DEFAULT 'user',
    public_key   VARCHAR        NOT NULL,
    timestamp    BIGINT         NOT NULL,
    CONSTRAINT check_timestamp CHECK (timestamp > 0),
    CONSTRAINT check_address_type CHECK (address_type IN ('user', 'hot', 'cold'))
);
CREATE INDEX IF NOT EXISTS idx_addresses_address ON addresses (address);
CREATE INDEX IF NOT EXISTS idx_addresses_address_type ON addresses (address_type);

CREATE TABLE IF NOT EXISTS tokens
(
    guid           VARCHAR PRIMARY KEY,
    token_address  VARCHAR  NOT NULL,
    decimals       SMALLINT NOT NULL DEFAULT 18,
    token_name     VARCHAR  NOT NULL,
    collect_amount UINT256  NOT NULL,
    cold_amount    UINT256  NOT NULL,
    timestamp      INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS tokens_timestamp ON tokens (timestamp);
CREATE INDEX IF NOT EXISTS tokens_token_address ON tokens (token_address);

CREATE TABLE IF NOT EXISTS balances
(
    guid          VARCHAR PRIMARY KEY,
    address       VARCHAR     NOT NULL,
    token_address VARCHAR     NOT NULL,
    address_type  VARCHAR(10) NOT NULL DEFAULT 'user',
    balance       UINT256     NOT NULL DEFAULT 0 CHECK (balance >= 0),
    lock_balance  UINT256     NOT NULL DEFAULT 0,
    timestamp     BIGINT      NOT NULL,
    CONSTRAINT check_timestamp CHECK (timestamp > 0),
    CONSTRAINT check_address_type CHECK (address_type IN ('user', 'hot', 'cold'))
);
CREATE INDEX IF NOT EXISTS idx_balances_address ON balances (address);
CREATE INDEX IF NOT EXISTS idx_balances_token_address ON balances (token_address);
CREATE INDEX IF NOT EXISTS idx_balances_address_type ON balances (address_type);

CREATE TABLE IF NOT EXISTS deposits
(
    guid                     VARCHAR PRIMARY KEY,
    timestamp                INTEGER  NOT NULL CHECK (timestamp > 0),
    status                   varchar  NOT NULL,
    confirms                 SMALLINT NOT NULL DEFAULT 0,

    block_hash               VARCHAR  NOT NULL,
    block_number             UINT256  NOT NULL CHECK (block_number > 0),
    hash                     VARCHAR  NOT NULL,
    tx_type                  VARCHAR  NOT NULL,

    from_address             VARCHAR  NOT NULL,
    to_address               VARCHAR  NOT NULL,
    amount                   UINT256  NOT NULL,

    gas_limit                INTEGER  NOT NULL,
    max_fee_per_gas          VARCHAR  NOT NULL,
    max_priority_fee_per_gas VARCHAR  NOT NULL,

    token_type               VARCHAR  NOT NULL,
    token_address            VARCHAR  NOT NULL,
    token_id                 VARCHAR  NOT NULL,
    token_meta               VARCHAR  NOT NULL,

    tx_sign_hex              VARCHAR  NOT NULL
);
CREATE INDEX IF NOT EXISTS deposits_hash ON deposits (hash);
CREATE INDEX IF NOT EXISTS deposits_timestamp ON deposits (timestamp);
CREATE INDEX IF NOT EXISTS deposits_from_address ON deposits (from_address);
CREATE INDEX IF NOT EXISTS deposits_to_address ON deposits (to_address);

CREATE TABLE IF NOT EXISTS withdraws
(
    guid                     VARCHAR PRIMARY KEY,
    timestamp                INTEGER NOT NULL CHECK (timestamp > 0),
    status                   VARCHAR NOT NULL,

    block_hash               VARCHAR NOT NULL,
    block_number             UINT256 NOT NULL CHECK (block_number > 0),
    hash                     VARCHAR NOT NULL,
    tx_type                  VARCHAR NOT NULL,

    from_address             VARCHAR NOT NULL,
    to_address               VARCHAR NOT NULL,
    amount                   UINT256 NOT NULL,

    gas_limit                INTEGER NOT NULL,
    max_fee_per_gas          VARCHAR NOT NULL,
    max_priority_fee_per_gas VARCHAR NOT NULL,

    token_type               VARCHAR NOT NULL,
    token_address            VARCHAR NOT NULL,
    token_id                 VARCHAR NOT NULL,
    token_meta               VARCHAR NOT NULL,

    tx_sign_hex              VARCHAR NOT NULL
);

CREATE INDEX IF NOT EXISTS withdraws_hash ON withdraws (hash);
CREATE INDEX IF NOT EXISTS withdraws_timestamp ON withdraws (timestamp);
CREATE INDEX IF NOT EXISTS withdraws_from_address ON withdraws (from_address);
CREATE INDEX IF NOT EXISTS withdraws_to_address ON withdraws (to_address);

CREATE TABLE IF NOT EXISTS internals
(
    guid                     VARCHAR PRIMARY KEY,
    timestamp                INTEGER NOT NULL CHECK (timestamp > 0),
    status                   VARCHAR NOT NULL,

    block_hash               VARCHAR NOT NULL,
    block_number             UINT256 NOT NULL CHECK (block_number > 0),
    hash                     VARCHAR NOT NULL,
    tx_type                  VARCHAR NOT NULL,

    from_address             VARCHAR NOT NULL,
    to_address               VARCHAR NOT NULL,
    amount                   UINT256 NOT NULL,

    gas_limit                INTEGER NOT NULL,
    max_fee_per_gas          VARCHAR NOT NULL,
    max_priority_fee_per_gas VARCHAR NOT NULL,

    token_type               VARCHAR NOT NULL,
    token_address            VARCHAR NOT NULL,
    token_id                 VARCHAR NOT NULL,
    token_meta               VARCHAR NOT NULL,

    tx_sign_hex              VARCHAR NOT NULL
);

CREATE INDEX IF NOT EXISTS internals_hash ON internals (hash);
CREATE INDEX IF NOT EXISTS internals_timestamp ON internals (timestamp);
CREATE INDEX IF NOT EXISTS internals_from_address ON internals (from_address);
CREATE INDEX IF NOT EXISTS internals_to_address ON internals (to_address);

