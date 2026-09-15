DROP INDEX IF EXISTS idx_tenants_pagarme_customer;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS pagarme_subscription_id,
    DROP COLUMN IF EXISTS pagarme_customer_id;
