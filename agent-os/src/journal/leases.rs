use super::Journal;
use crate::storage::{LeaseManager, Reservation, ReservationReceipt};
use anyhow::{ensure, Context, Result};
use rusqlite::{params, OptionalExtension, Transaction};

impl Journal {
    fn acquire_lease(
        &self,
        transaction: &Transaction<'_>,
        resource: &str,
        holder: &str,
        now: i64,
        ttl: i64,
    ) -> Result<Option<u64>> {
        ensure!(
            !resource.is_empty() && !holder.is_empty() && now >= 0 && ttl > 0,
            "invalid lease request"
        );
        let expiry = now.checked_add(ttl).context("lease expiry overflow")?;
        let current: Option<(u64, i64, String)> = transaction
            .query_row(
                "SELECT token,expires,status FROM leases WHERE resource=?1",
                [resource],
                |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?)),
            )
            .optional()?;
        let token = if let Some((token, expires, status)) = current {
            if status == "reconciling" || (status == "active" && expires > now) {
                return Ok(None);
            }
            let unresolved:bool=transaction.query_row("SELECT EXISTS(SELECT 1 FROM actions WHERE resource=?1 AND lease_token=?2 AND status IN ('pending','uncertain'))",params![resource,token],|row|row.get(0))?;
            if status != "released" && unresolved {
                transaction.execute(
                    "UPDATE leases SET status='reconciling' WHERE resource=?1 AND token=?2",
                    params![resource, token],
                )?;
                return Ok(None);
            }
            token
                .checked_add(1)
                .filter(|t| *t <= i64::MAX as u64)
                .context("lease token exhausted")?
        } else {
            1
        };
        transaction.execute("INSERT INTO leases(resource,holder,token,expires,status) VALUES (?1,?2,?3,?4,'active') ON CONFLICT(resource) DO UPDATE SET holder=excluded.holder,token=excluded.token,expires=excluded.expires,status='active'",params![resource,holder,token,expiry])?;
        Ok(Some(token))
    }
    pub fn lease_acquire(&self, resource: &str, holder: &str, now: i64, ttl: i64) -> Result<u64> {
        let transaction = self.transaction()?;
        let token = self.acquire_lease(&transaction, resource, holder, now, ttl)?;
        transaction.commit()?;
        token.context("resource is leased or requires reconciliation")
    }
    pub fn lease_release(&self, resource: &str, holder: &str, token: u64) -> Result<()> {
        let transaction = self.transaction()?;
        let unresolved:bool=transaction.query_row("SELECT EXISTS(SELECT 1 FROM actions WHERE resource=?1 AND lease_token=?2 AND status IN ('pending','uncertain'))",params![resource,token],|row|row.get(0))?;
        ensure!(
            !unresolved,
            "external outcome requires reconciliation before release"
        );
        let changed=transaction.execute("UPDATE leases SET status='released' WHERE resource=?1 AND holder=?2 AND token=?3 AND status='active'",params![resource,holder,token])?;
        ensure!(changed == 1, "stale or unknown lease holder");
        transaction.commit()?;
        Ok(())
    }
    /// Explicit operator acknowledgment that an uncertain external operation has been reconciled.
    /// This releases only the exact reservation; it neither retries nor changes its outcome.
    pub fn reconcile_lease(&self, resource: &str, holder: &str, token: u64) -> Result<()> {
        let changed=self.connection.execute("UPDATE leases SET status='released' WHERE resource=?1 AND holder=?2 AND token=?3 AND status='reconciling' AND EXISTS(SELECT 1 FROM actions WHERE resource=?1 AND lease_token=?3 AND run=?2 AND status='uncertain') AND NOT EXISTS(SELECT 1 FROM actions WHERE resource=?1 AND lease_token=?3 AND status='pending')",params![resource,holder,token])?;
        ensure!(changed == 1, "stale or non-reconciling reservation");
        Ok(())
    }
}
impl LeaseManager for Journal {
    fn reserve(&self, request: Reservation<'_>) -> Result<Option<ReservationReceipt>> {
        ensure!(
            request.effect["capability"]
                .as_str()
                .is_some_and(|value| !value.is_empty()),
            "reservation effect requires a capability"
        );
        self.ensure_task_mailbox(request.run)?;
        let transaction = self.transaction()?;
        let Some(token) = self.acquire_lease(
            &transaction,
            request.resource,
            request.run,
            request.now,
            request.ttl,
        )?
        else {
            transaction.commit()?;
            return Ok(None);
        };
        let action = self
            .commit_transition(
                &transaction,
                request.run,
                request.state,
                request.signal,
                Some(request.effect),
                "running",
            )?
            .context("missing reserved action")?;
        transaction.execute(
            "UPDATE actions SET resource=?1,lease_token=?2 WHERE id=?3",
            params![request.resource, token, action],
        )?;
        transaction.commit()?;
        Ok(Some(ReservationReceipt { action, token }))
    }
}
