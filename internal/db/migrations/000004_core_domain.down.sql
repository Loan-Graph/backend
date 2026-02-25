DROP INDEX IF EXISTS idx_outbox_jobs_status_available_at;
DROP TABLE IF EXISTS outbox_jobs;

DROP INDEX IF EXISTS idx_events_event;
DROP INDEX IF EXISTS idx_events_processed;
DROP TABLE IF EXISTS chain_events;

DROP INDEX IF EXISTS idx_pools_status;
DROP INDEX IF EXISTS idx_pools_lender;
DROP TABLE IF EXISTS pools;

DROP TABLE IF EXISTS passport_cache;

DROP INDEX IF EXISTS idx_repayments_loan;
DROP TABLE IF EXISTS repayments;

DROP INDEX IF EXISTS idx_loans_maturity;
DROP INDEX IF EXISTS idx_loans_status;
DROP INDEX IF EXISTS idx_loans_borrower;
DROP INDEX IF EXISTS idx_loans_lender;
DROP TABLE IF EXISTS loans;

DROP INDEX IF EXISTS idx_borrowers_hash;
DROP TABLE IF EXISTS borrowers;

DROP TABLE IF EXISTS lenders;
