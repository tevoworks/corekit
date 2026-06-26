-- Add resolved_ips column for DNS pinning (SSRF prevention).
-- The resolved IPs are stored at webhook creation time and validated
-- at dispatch time to prevent DNS rebinding attacks.

ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS resolved_ips TEXT[] DEFAULT '{}';
