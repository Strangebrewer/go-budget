ALTER TABLE bills DROP CONSTRAINT bills_source_id_fkey;
ALTER TABLE bills ADD CONSTRAINT bills_source_id_fkey
    FOREIGN KEY (source_id) REFERENCES accounts(id) ON DELETE CASCADE;

ALTER TABLE transactions DROP CONSTRAINT transactions_source_id_fkey;
ALTER TABLE transactions ADD CONSTRAINT transactions_source_id_fkey
    FOREIGN KEY (source_id) REFERENCES accounts(id) ON DELETE SET NULL;

ALTER TABLE transactions DROP CONSTRAINT transactions_destination_id_fkey;
ALTER TABLE transactions ADD CONSTRAINT transactions_destination_id_fkey
    FOREIGN KEY (destination_id) REFERENCES accounts(id) ON DELETE SET NULL;
