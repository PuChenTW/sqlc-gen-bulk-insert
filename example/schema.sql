-- example schema for sqlc-gen-bulk-insert

CREATE TABLE users (
    id         BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    age        INT,                        -- nullable: maps to sql.NullInt32
    created_at DATETIME     NOT NULL
);

CREATE TABLE products (
    id          BIGINT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    sku         VARCHAR(100)    NOT NULL,
    title       VARCHAR(255)    NOT NULL,
    price_cents INT             NOT NULL,
    in_stock    TINYINT(1)      NOT NULL DEFAULT 1,
    created_at  DATETIME        NOT NULL
);
