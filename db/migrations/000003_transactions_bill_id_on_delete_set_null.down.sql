ALTER TABLE transactions DROP CONSTRAINT transactions_bill_id_fkey;
ALTER TABLE transactions ADD CONSTRAINT transactions_bill_id_fkey
    FOREIGN KEY (bill_id) REFERENCES bills(id);
