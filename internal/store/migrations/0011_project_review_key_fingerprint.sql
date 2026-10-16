-- Legacy attestations remain explicitly unverified: their fingerprints stay
-- empty and are never inferred from reviewer_id or historical signatures.
ALTER TABLE project_reviews
    ADD COLUMN reviewer_key_fingerprint TEXT NOT NULL DEFAULT '';
