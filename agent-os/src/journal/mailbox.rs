use super::Journal;
use crate::storage::{Mailbox, Signal};
use anyhow::{ensure, Context, Result};
use rusqlite::Transaction;
use rusqlite::{params, OptionalExtension};
use serde_json::Value;

impl Journal {
    pub fn register_mailbox(
        &self,
        consumer: &str,
        run: &str,
        filter: &[String],
        coalescing: &[String],
    ) -> Result<()> {
        ensure!(
            !consumer.starts_with("task:"),
            "task mailbox namespace is reserved"
        );
        ensure!(
            !filter.is_empty() && coalescing.iter().all(|kind| filter.contains(kind)),
            "invalid filtered subscription"
        );
        self.register_subscription(consumer, run, filter, coalescing, 0)
    }

    fn register_subscription(
        &self,
        consumer: &str,
        run: &str,
        filter: &[String],
        coalescing: &[String],
        cursor: i64,
    ) -> Result<()> {
        ensure!(
            !consumer.is_empty()
                && filter.iter().all(|kind| !kind.is_empty())
                && coalescing.iter().all(|kind| !kind.is_empty()),
            "invalid mailbox subscription"
        );
        let transaction = self.transaction()?;
        let existing: Option<(String, String, String)> = transaction
            .query_row(
                "SELECT run,filter,coalescing FROM mailboxes WHERE consumer=?1",
                [consumer],
                |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?)),
            )
            .optional()?;
        let filter = serde_json::to_string(filter)?;
        let coalescing = serde_json::to_string(coalescing)?;
        if let Some(existing) = existing {
            ensure!(
                existing == (run.to_string(), filter, coalescing),
                "mailbox subscription changed"
            );
        } else {
            transaction.execute(
                "INSERT INTO mailboxes(consumer,run,filter,coalescing,cursor) VALUES (?1,?2,?3,?4,?5)",
                params![consumer, run, filter, coalescing, cursor],
            )?;
        }
        transaction.commit()?;
        Ok(())
    }

    fn mailbox_pending(&self, consumer: &str) -> Result<(Vec<Signal>, Vec<String>)> {
        let (run, filter, coalescing, cursor): (String, String, String, i64) =
            self.connection.query_row(
                "SELECT run,filter,coalescing,cursor FROM mailboxes WHERE consumer=?1",
                [consumer],
                |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?, row.get(3)?)),
            )?;
        let filter_kinds: Vec<String> = serde_json::from_str(&filter)?;
        let coalescing: Vec<String> = serde_json::from_str(&coalescing)?;
        // signals_run bounds the scan to this subscription's unconsumed range.
        // The composite mailbox_ack primary key excludes out-of-order ACKs without
        // reading any historical payload or issuing one query per signal.
        let mut statement=self.connection.prepare(
            "SELECT s.id,s.kind,s.payload,s.action,a.artifact
             FROM signals s LEFT JOIN actions a ON a.id=s.action
             WHERE s.run=?1 AND s.id>?2
               AND NOT EXISTS(SELECT 1 FROM mailbox_ack ack WHERE ack.consumer=?3 AND ack.signal=s.id)
               AND (?5 OR s.kind IN (SELECT value FROM json_each(?4)))
             ORDER BY s.id")?;
        let rows = statement.query_map(
            params![run, cursor, consumer, filter, filter_kinds.is_empty()],
            |row| {
                Ok((
                    row.get::<_, i64>(0)?,
                    row.get::<_, String>(1)?,
                    row.get::<_, String>(2)?,
                    row.get::<_, Option<i64>>(3)?,
                    row.get::<_, Option<String>>(4)?,
                ))
            },
        )?;
        let pending = rows
            .map(|row| {
                let (id, kind, payload, action, artifact) = row?;
                Ok(Signal {
                    id,
                    kind,
                    payload: serde_json::from_str(&payload)?,
                    action,
                    generation: action,
                    artifact,
                })
            })
            .collect::<Result<Vec<_>>>()?;
        Ok((pending, coalescing))
    }

    pub fn mailbox_next(&self, consumer: &str) -> Result<Option<Signal>> {
        let (pending, coalescing) = self.mailbox_pending(consumer)?;
        Ok(select_delivery(&pending, &coalescing).cloned())
    }

    pub fn mailbox_ack(&self, consumer: &str, signal_id: i64) -> Result<()> {
        ensure!(
            !consumer.starts_with("task:"),
            "task mailbox acknowledgment requires an atomic checkpoint"
        );
        let transaction = self.transaction()?;
        self.ack_mailbox(&transaction, consumer, signal_id)?;
        transaction.commit()?;
        Ok(())
    }

    pub(super) fn ack_mailbox(
        &self,
        transaction: &Transaction<'_>,
        consumer: &str,
        signal_id: i64,
    ) -> Result<i64> {
        let (pending, coalescing) = self.mailbox_pending(consumer)?;
        let first = pending.first().context("mailbox has no pending delivery")?;
        let delivered = pending
            .iter()
            .find(|signal| signal.id == signal_id)
            .context("unknown mailbox delivery")?;
        // A newer level notification may arrive while the consumer processes an older one.
        // Acknowledge only through the delivered occurrence, preserving the newer wake-up.
        ensure!(
            delivered.id == first.id
                || (coalescing.contains(&first.kind) && same_coalescing_key(first, delivered)),
            "ack bypasses an earlier queue delivery"
        );
        let acknowledged = |signal: &&Signal| {
            signal.id == delivered.id
                || (coalescing.contains(&delivered.kind)
                    && same_coalescing_key(signal, delivered)
                    && signal.id < delivered.id)
        };
        for signal in pending.iter().filter(acknowledged) {
            transaction.execute(
                "INSERT INTO mailbox_ack(consumer,signal) VALUES (?1,?2)",
                params![consumer, signal.id],
            )?;
        }
        // IMMEDIATE excludes concurrent writers; the frontier follows the first
        // retained occurrence, never a later coalesced occurrence past queue work.
        let cursor = pending
            .iter()
            .find(|signal| !acknowledged(signal))
            .map(|signal| signal.id - 1)
            .unwrap_or_else(|| pending.last().map_or(0, |signal| signal.id));
        transaction.execute(
            "UPDATE mailboxes SET cursor=MAX(cursor,?1) WHERE consumer=?2",
            params![cursor, consumer],
        )?;
        transaction.execute(
            "DELETE FROM mailbox_ack WHERE consumer=?1 AND signal<=?2",
            params![consumer, cursor],
        )?;
        Ok(cursor)
    }
    pub fn prepare_mailbox(&self, run: &str, coalescing: &[String]) -> Result<()> {
        let current = self.run(run)?;
        self.register_subscription(&task_consumer(run), run, &[], coalescing, current.cursor)
    }
    pub(super) fn ensure_task_mailbox(&self, run: &str) -> Result<()> {
        let exists: bool = self.connection.query_row(
            "SELECT EXISTS(SELECT 1 FROM mailboxes WHERE consumer=?1)",
            [task_consumer(run)],
            |row| row.get(0),
        )?;
        if !exists {
            self.prepare_mailbox(run, &[])?;
        }
        Ok(())
    }
}

fn same_coalescing_key(left: &Signal, right: &Signal) -> bool {
    left.kind == right.kind && left.payload.get("target") == right.payload.get("target")
}

fn select_delivery<'a>(pending: &'a [Signal], coalescing: &[String]) -> Option<&'a Signal> {
    let first = pending.first()?;
    if coalescing.contains(&first.kind) {
        pending
            .iter()
            .rev()
            .find(|signal| same_coalescing_key(signal, first))
    } else {
        Some(first)
    }
}

pub(super) fn task_consumer(run: &str) -> String {
    format!("task:{run}")
}
impl Mailbox for Journal {
    fn receive(&self, id: &str) -> Result<Option<Signal>> {
        self.ensure_task_mailbox(id)?;
        self.mailbox_next(&task_consumer(id))
    }
    fn commit(&self, id: &str, state: &Value, signal: i64, status: &str) -> Result<()> {
        self.ensure_task_mailbox(id)?;
        let transaction = self.transaction()?;
        self.commit_transition(&transaction, id, state, signal, None, status)?;
        transaction.commit()?;
        Ok(())
    }
}
