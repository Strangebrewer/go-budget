UPDATE accounts SET type = 'asset' WHERE type = 'checking';
UPDATE accounts SET type = 'debt' WHERE type = 'credit';
ALTER TABLE accounts ADD CONSTRAINT accounts_type_check CHECK (type IN ('asset', 'debt'));
