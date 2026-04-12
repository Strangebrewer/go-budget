CREATE TABLE accounts (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL,
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    balance     int         NOT NULL DEFAULT 0,
    owner       text        NOT NULL DEFAULT 'mine',
    status      text        NOT NULL DEFAULT 'active',
    type        text        NOT NULL DEFAULT 'debt',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX accounts_user_id_idx ON accounts(user_id);

CREATE TABLE categories (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL,
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX categories_user_id_idx ON categories(user_id);

CREATE TABLE bills (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL,
    source_id   uuid        NOT NULL REFERENCES accounts(id),
    category_id uuid        REFERENCES categories(id),
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    due_day     int         NOT NULL DEFAULT 1,
    owner       text        NOT NULL DEFAULT 'mine',
    shared      boolean     NOT NULL DEFAULT false,
    status      text        NOT NULL DEFAULT 'active',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX bills_user_id_idx ON bills(user_id);

CREATE TABLE transactions (
    id             uuid        PRIMARY KEY,
    user_id        uuid        NOT NULL,
    source_id      uuid        REFERENCES accounts(id),
    destination_id uuid        REFERENCES accounts(id),
    bill_id        uuid        REFERENCES bills(id),
    category_id    uuid        REFERENCES categories(id),
    amount         int         NOT NULL,
    bill_month     text,
    date           date        NOT NULL,
    description    text        NOT NULL DEFAULT '',
    income         boolean     NOT NULL DEFAULT false,
    owner          text        NOT NULL DEFAULT 'mine',
    shared         boolean     NOT NULL DEFAULT false,
    type           text        NOT NULL DEFAULT 'expense',
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX transactions_user_id_idx ON transactions(user_id);
CREATE INDEX transactions_bill_month_idx ON transactions(bill_month);
