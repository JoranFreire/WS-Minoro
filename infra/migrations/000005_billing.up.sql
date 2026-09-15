-- Billing: link a tenant to its Pagar.me customer/subscription.
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS pagarme_customer_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS pagarme_subscription_id VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_tenants_pagarme_customer
    ON tenants (pagarme_customer_id)
    WHERE pagarme_customer_id IS NOT NULL;
