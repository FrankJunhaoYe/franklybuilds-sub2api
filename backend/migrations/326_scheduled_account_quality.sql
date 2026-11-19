ALTER TABLE accounts ADD COLUMN quality_holds jsonb;
ALTER TABLE scheduled_test_plans
    ADD COLUMN quality_rule jsonb,
    ADD COLUMN quality_state jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN quality_lease_until timestamptz;
ALTER TABLE scheduled_test_results ADD COLUMN quality_evidence jsonb;

-- Aggregate the remaining rule owners under the account lock. A successful
-- retest, plan edit or deletion can only release this plan's contribution.
CREATE FUNCTION refresh_scheduled_quality_holds() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    target bigint;
    holds jsonb;
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.quality_rule IS NULL THEN RETURN NULL; END IF;
        target := OLD.account_id;
    ELSE
        IF NEW.quality_rule IS NOT DISTINCT FROM OLD.quality_rule
            AND NEW.enabled = OLD.enabled
            AND NEW.quality_state = OLD.quality_state THEN RETURN NULL; END IF;
        target := NEW.account_id;
    END IF;
    PERFORM id FROM accounts WHERE id = target FOR UPDATE;
    SELECT jsonb_object_agg(scope, true) INTO holds
    FROM (
        SELECT DISTINCT CASE p.quality_rule->>'action'
            WHEN 'model' THEN 'model:' || s.key
            WHEN 'group' THEN 'group:' || (p.quality_rule->>'group_id')
            WHEN 'account' THEN 'account' END AS scope
        FROM scheduled_test_plans p
        CROSS JOIN LATERAL jsonb_each(p.quality_state) s
        WHERE p.account_id = target AND p.enabled
          AND p.quality_rule->>'action' IN ('model','group','account')
          AND s.value->>'held' = 'true'
    ) active;
    UPDATE accounts SET quality_holds = holds
    WHERE id = target AND quality_holds IS DISTINCT FROM holds;
    IF FOUND THEN
        INSERT INTO scheduler_outbox(event_type, account_id) VALUES ('account_changed', target);
    END IF;
    RETURN NULL;
END $$;
CREATE TRIGGER scheduled_quality_holds_updated
AFTER UPDATE OF enabled, quality_rule, quality_state OR DELETE ON scheduled_test_plans
FOR EACH ROW EXECUTE FUNCTION refresh_scheduled_quality_holds();
