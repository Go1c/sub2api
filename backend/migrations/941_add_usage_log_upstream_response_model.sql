-- Record the model declared by a successful upstream response, plus whether it
-- mismatched the model this gateway actually sent. Used by billing_model_source=response_model.
ALTER TABLE usage_logs
 ADD COLUMN IF NOT EXISTS upstream_response_model VARCHAR(200),
 ADD COLUMN IF NOT EXISTS upstream_model_mismatch BOOLEAN;
