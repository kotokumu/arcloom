//! Durable task history, content-addressed artifacts and owner-scoped knowledge.

use anyhow::{ensure, Context, Result};
use rusqlite::{params, Connection, OptionalExtension, Transaction, TransactionBehavior};
use serde::Serialize;
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Duration;

pub struct Journal {
    connection: Connection,
    cas: PathBuf,
}

#[derive(Debug, Serialize)]
pub struct Run {
    pub id: String,
    pub binding: String,
    pub intent: Value,
    pub state: Value,
    pub status: String,
    pub actions: u64,
    pub cursor: i64,
}

#[derive(Debug, Clone, Serialize)]
pub struct Signal {
    pub id: i64,
    pub kind: String,
    pub payload: Value,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub action: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub generation: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub artifact: Option<String>,
}

impl Journal {
    pub fn open(dir: &Path) -> Result<Self> {
        fs::create_dir_all(dir)?;
        let cas = dir.join("cas");
        fs::create_dir_all(&cas)?;
        let connection = Connection::open(dir.join("journal.sqlite"))?;
        connection.busy_timeout(Duration::from_secs(5))?;
        connection.pragma_update(None, "foreign_keys", "ON")?;
        connection.pragma_update(None, "journal_mode", "WAL")?;
        connection.pragma_update(None, "synchronous", "FULL")?;
        let transaction = Transaction::new_unchecked(&connection, TransactionBehavior::Immediate)?;
        let version: i64 =
            transaction.pragma_query_value(None, "user_version", |row| row.get(0))?;
        ensure!(
            (0..=2).contains(&version),
            "unsupported journal schema version {version}"
        );
        transaction.execute_batch(
            "CREATE TABLE IF NOT EXISTS runs (
                id TEXT PRIMARY KEY, binding TEXT NOT NULL, intent TEXT NOT NULL,
                state TEXT NOT NULL, status TEXT NOT NULL, actions INTEGER NOT NULL DEFAULT 0,
                cursor INTEGER NOT NULL DEFAULT 0);
             CREATE TABLE IF NOT EXISTS actions (
                id INTEGER PRIMARY KEY AUTOINCREMENT, run TEXT NOT NULL REFERENCES runs(id),
                effect TEXT NOT NULL, status TEXT NOT NULL);
             CREATE TABLE IF NOT EXISTS signals (
                id INTEGER PRIMARY KEY AUTOINCREMENT, run TEXT NOT NULL REFERENCES runs(id),
                kind TEXT NOT NULL, payload TEXT NOT NULL, action INTEGER REFERENCES actions(id));
             CREATE INDEX IF NOT EXISTS signals_run ON signals(run,id);
             CREATE TABLE IF NOT EXISTS knowledge (id TEXT PRIMARY KEY, body TEXT NOT NULL);
             CREATE TABLE IF NOT EXISTS leases (
                resource TEXT PRIMARY KEY, holder TEXT NOT NULL, token INTEGER NOT NULL,
                expires INTEGER NOT NULL, status TEXT NOT NULL);
             CREATE TABLE IF NOT EXISTS mailboxes (
                consumer TEXT PRIMARY KEY, run TEXT NOT NULL REFERENCES runs(id),
                filter TEXT NOT NULL, coalescing TEXT NOT NULL, cursor INTEGER NOT NULL DEFAULT 0);
             CREATE TABLE IF NOT EXISTS mailbox_ack (
                consumer TEXT NOT NULL REFERENCES mailboxes(consumer), signal INTEGER NOT NULL REFERENCES signals(id),
                PRIMARY KEY(consumer,signal));",
        )?;
        if version < 2 {
            transaction.execute_batch(
                "ALTER TABLE actions ADD COLUMN artifact TEXT; PRAGMA user_version = 2;",
            )?;
        }
        transaction.commit()?;
        Ok(Self { connection, cas })
    }

    fn transaction(&self) -> Result<Transaction<'_>> {
        Ok(Transaction::new_unchecked(
            &self.connection,
            TransactionBehavior::Immediate,
        )?)
    }

    pub fn create_run(&self, id: &str, binding: &str, intent: &Value) -> Result<()> {
        ensure!(
            !id.is_empty() && !binding.is_empty(),
            "run ID and binding must not be empty"
        );
        let transaction = self.transaction()?;
        transaction.execute(
            "INSERT INTO runs(id,binding,intent,state,status) VALUES (?1,?2,?3,'null','ready')",
            params![id, binding, intent.to_string()],
        )?;
        transaction.execute(
            "INSERT INTO signals(run,kind,payload) VALUES (?1,'start',?2)",
            params![id, intent.to_string()],
        )?;
        transaction.commit()?;
        Ok(())
    }

    pub fn run(&self, id: &str) -> Result<Run> {
        let (binding, intent, state, status, actions, cursor): (
            String,
            String,
            String,
            String,
            u64,
            i64,
        ) = self
            .connection
            .query_row(
                "SELECT binding,intent,state,status,actions,cursor FROM runs WHERE id=?1",
                [id],
                |row| {
                    Ok((
                        row.get(0)?,
                        row.get(1)?,
                        row.get(2)?,
                        row.get(3)?,
                        row.get(4)?,
                        row.get(5)?,
                    ))
                },
            )
            .with_context(|| format!("unknown run {id}"))?;
        Ok(Run {
            id: id.to_string(),
            binding,
            intent: serde_json::from_str(&intent)?,
            state: serde_json::from_str(&state)?,
            status,
            actions,
            cursor,
        })
    }

    pub fn signals(&self, run: &str) -> Result<Vec<Signal>> {
        self.run(run)?;
        let mut statement = self
            .connection
            .prepare("SELECT s.id,s.kind,s.payload,s.action,a.artifact FROM signals s LEFT JOIN actions a ON a.id=s.action WHERE s.run=?1 ORDER BY s.id")?;
        let rows = statement.query_map([run], |row| {
            Ok((
                row.get::<_, i64>(0)?,
                row.get::<_, String>(1)?,
                row.get::<_, String>(2)?,
                row.get::<_, Option<i64>>(3)?,
                row.get::<_, Option<String>>(4)?,
            ))
        })?;
        rows.map(|row| {
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
        .collect()
    }

    pub fn next_signal(&self, run: &str) -> Result<Option<Signal>> {
        let cursor = self.run(run)?.cursor;
        Ok(self
            .signals(run)?
            .into_iter()
            .find(|signal| signal.id > cursor))
    }

    pub fn transition(
        &self,
        run: &str,
        state: &Value,
        signal_id: i64,
        effect: Option<&Value>,
        status: &str,
    ) -> Result<Option<i64>> {
        ensure!(!status.is_empty(), "empty run status");
        let transaction = self.transaction()?;
        let next: Option<i64> = transaction.query_row(
            "SELECT MIN(s.id) FROM signals s JOIN runs r ON r.id=s.run WHERE s.run=?1 AND s.id>r.cursor",
            [run], |row| row.get(0),
        )?;
        ensure!(
            next == Some(signal_id),
            "signal is not the next unacknowledged input of this run"
        );
        let pending: bool = transaction.query_row(
            "SELECT EXISTS(SELECT 1 FROM actions WHERE run=?1 AND status='pending')",
            [run],
            |row| row.get(0),
        )?;
        ensure!(!pending, "an action is already pending");
        let mut action = None;
        if let Some(effect) = effect {
            transaction.execute(
                "INSERT INTO actions(run,effect,status) VALUES (?1,?2,'pending')",
                params![run, effect.to_string()],
            )?;
            action = Some(transaction.last_insert_rowid());
        }
        transaction.execute(
            "UPDATE runs SET state=?1,cursor=?2,status=?3,actions=actions+?4 WHERE id=?5",
            params![
                state.to_string(),
                signal_id,
                status,
                i64::from(effect.is_some()),
                run
            ],
        )?;
        if status == "budget_hit" || status == "denied" {
            transaction.execute(
                "INSERT INTO signals(run,kind,payload) VALUES (?1,?2,'{}')",
                params![run, status],
            )?;
        }
        transaction.commit()?;
        Ok(action)
    }

    pub fn complete_action(
        &self,
        run: &str,
        action: i64,
        kind: &str,
        payload: &Value,
    ) -> Result<()> {
        ensure!(!kind.is_empty(), "signal kind must not be empty");
        let transaction = self.transaction()?;
        let changed = transaction.execute(
            "UPDATE actions SET status='completed' WHERE id=?1 AND run=?2 AND status='pending'",
            params![action, run],
        )?;
        ensure!(
            changed == 1,
            "unknown, stale, uncertain or already completed action"
        );
        // Persist and verify immutable bytes before committing their journal reference.
        // A later SQL failure can leave an unreferenced CAS blob, never a missing referenced blob.
        let artifact = self.put_artifact(&serde_json::to_vec(payload)?)?;
        transaction.execute(
            "UPDATE actions SET artifact=?1 WHERE id=?2",
            params![artifact, action],
        )?;
        transaction.execute(
            "INSERT INTO signals(run,kind,payload,action) VALUES (?1,?2,?3,?4)",
            params![run, kind, payload.to_string(), action],
        )?;
        transaction.execute("UPDATE runs SET status='ready' WHERE id=?1", [run])?;
        transaction.commit()?;
        Ok(())
    }

    pub fn recover(&self, run: &str) -> Result<bool> {
        self.run(run)?;
        let transaction = self.transaction()?;
        let pending: Option<i64> = transaction
            .query_row(
                "SELECT id FROM actions WHERE run=?1 AND status='pending' ORDER BY id LIMIT 1",
                [run],
                |row| row.get(0),
            )
            .optional()?;
        if let Some(action) = pending {
            transaction.execute(
                "UPDATE actions SET status='uncertain' WHERE id=?1",
                [action],
            )?;
            transaction.execute("UPDATE runs SET status='uncertain' WHERE id=?1", [run])?;
            transaction.execute(
                "INSERT INTO signals(run,kind,payload,action) VALUES (?1,'stall',?2,?3)",
                params![
                    run,
                    json!({"reason":"external_outcome_uncertain","replay":false}).to_string(),
                    action
                ],
            )?;
        }
        transaction.commit()?;
        Ok(pending.is_some())
    }

    pub fn put_artifact(&self, bytes: &[u8]) -> Result<String> {
        let hash = format!("{:x}", Sha256::digest(bytes));
        let destination = self.cas.join(&hash);
        if destination.exists() {
            self.get_artifact(&hash)?;
            return Ok(hash);
        }
        static SEQUENCE: AtomicU64 = AtomicU64::new(0);
        let temporary = self.cas.join(format!(
            ".put-{}-{}",
            std::process::id(),
            SEQUENCE.fetch_add(1, Ordering::Relaxed)
        ));
        let result = (|| -> Result<()> {
            let mut file = OpenOptions::new()
                .create_new(true)
                .write(true)
                .open(&temporary)?;
            file.write_all(bytes)?;
            file.sync_all()?;
            match fs::hard_link(&temporary, &destination) {
                Ok(()) => (),
                Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                    self.get_artifact(&hash)?;
                }
                Err(error) => return Err(error.into()),
            }
            fs::File::open(&self.cas)?.sync_all()?;
            Ok(())
        })();
        let _ = fs::remove_file(&temporary);
        result?;
        Ok(hash)
    }

    pub fn get_artifact(&self, hash: &str) -> Result<Vec<u8>> {
        ensure!(
            hash.len() == 64
                && hash
                    .bytes()
                    .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b)),
            "invalid artifact hash"
        );
        let bytes = fs::read(self.cas.join(hash)).context("artifact is unavailable")?;
        ensure!(
            format!("{:x}", Sha256::digest(&bytes)) == hash,
            "artifact checksum mismatch"
        );
        Ok(bytes)
    }

    pub fn add_knowledge(&self, node: &Value) -> Result<()> {
        let id = node["id"]
            .as_str()
            .filter(|v| !v.is_empty())
            .context("knowledge id must be nonempty")?;
        let kind = node["kind"].as_str().context("knowledge kind required")?;
        ensure!(
            ["decision", "finding", "hypothesis", "disproved"].contains(&kind),
            "invalid knowledge kind"
        );
        let status = node["status"]
            .as_str()
            .context("knowledge status required")?;
        ensure!(
            ["draft", "stable", "deprecated"].contains(&status),
            "invalid knowledge status"
        );
        let trust = node["trust"].as_str().context("knowledge trust required")?;
        ensure!(
            ["unverified", "machine", "human"].contains(&trust),
            "invalid knowledge trust"
        );
        if status == "draft" || !node["stale_after"].is_null() {
            ensure!(
                node["stale_after"].as_i64().is_some_and(|time| time > 0),
                "drafts require a positive expiry timestamp"
            );
        }
        self.validate_sources(node)?;
        self.connection.execute(
            "INSERT INTO knowledge(id,body) VALUES (?1,?2) ON CONFLICT(id) DO UPDATE SET body=excluded.body",
            params![id, node.to_string()],
        )?;
        Ok(())
    }

    fn validate_sources(&self, node: &Value) -> Result<()> {
        for source in node["sources"]
            .as_array()
            .context("knowledge sources must be an array")?
        {
            let hash = source
                .as_str()
                .or_else(|| source["sha256"].as_str())
                .context("source must reference an artifact hash")?;
            self.get_artifact(hash)?;
        }
        Ok(())
    }

    pub fn context(&self, now: i64) -> Result<Value> {
        let mut confirmed = Vec::new();
        let mut provisional = Vec::new();
        let mut statement = self
            .connection
            .prepare("SELECT body FROM knowledge ORDER BY id")?;
        let rows = statement.query_map([], |row| row.get::<_, String>(0))?;
        for row in rows {
            let node: Value = serde_json::from_str(&row?)?;
            if node["status"] == "deprecated"
                || node["stale_after"]
                    .as_i64()
                    .is_some_and(|expiry| expiry <= now)
            {
                continue;
            }
            self.validate_sources(&node)?;
            if node["status"] == "stable" && node["trust"] == "human" {
                confirmed.push(node);
            } else {
                provisional.push(node);
            }
        }
        Ok(json!({"confirmed":confirmed,"provisional":provisional}))
    }

    pub fn lease_acquire(&self, resource: &str, holder: &str, now: i64, ttl: i64) -> Result<u64> {
        ensure!(
            !resource.is_empty() && !holder.is_empty() && ttl > 0 && now >= 0,
            "invalid lease request"
        );
        let expiry = now.checked_add(ttl).context("lease expiry overflow")?;
        let transaction = self.transaction()?;
        let current: Option<(u64, i64, String)> = transaction
            .query_row(
                "SELECT token,expires,status FROM leases WHERE resource=?1",
                [resource],
                |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?)),
            )
            .optional()?;
        let token = if let Some((token, expires, status)) = current {
            ensure!(status == "released" || expires <= now, "resource is leased");
            token
                .checked_add(1)
                .filter(|t| *t <= i64::MAX as u64)
                .context("lease token exhausted")?
        } else {
            1
        };
        transaction.execute(
            "INSERT INTO leases(resource,holder,token,expires,status) VALUES (?1,?2,?3,?4,'active')
             ON CONFLICT(resource) DO UPDATE SET holder=excluded.holder,token=excluded.token,expires=excluded.expires,status='active'",
            params![resource, holder, token, expiry],
        )?;
        transaction.commit()?;
        Ok(token)
    }

    pub fn lease_release(&self, resource: &str, holder: &str, token: u64) -> Result<()> {
        let changed = self.connection.execute(
            "UPDATE leases SET status='released' WHERE resource=?1 AND holder=?2 AND token=?3 AND status='active'",
            params![resource, holder, token],
        )?;
        ensure!(changed == 1, "stale or unknown lease holder");
        Ok(())
    }
    /// Append one immutable occurrence. Duplicate occurrences are distinct inputs.
    pub fn append_signal(&self, run: &str, kind: &str, payload: &Value) -> Result<i64> {
        ensure!(!kind.is_empty(), "signal kind must not be empty");
        self.connection.execute(
            "INSERT INTO signals(run,kind,payload) VALUES (?1,?2,?3)",
            params![run, kind, payload.to_string()],
        )?;
        Ok(self.connection.last_insert_rowid())
    }

    /// Register a durable subscription. Existing delivery contracts cannot change silently.
    pub fn register_mailbox(
        &self,
        consumer: &str,
        run: &str,
        filter: &[String],
        coalescing: &[String],
    ) -> Result<()> {
        ensure!(
            !consumer.is_empty()
                && !filter.is_empty()
                && filter.iter().all(|kind| !kind.is_empty()),
            "invalid mailbox subscription"
        );
        ensure!(
            coalescing.iter().all(|kind| filter.contains(kind)),
            "coalescing kinds must be subscribed"
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
                "INSERT INTO mailboxes(consumer,run,filter,coalescing) VALUES (?1,?2,?3,?4)",
                params![consumer, run, filter, coalescing],
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
        let filter: Vec<String> = serde_json::from_str(&filter)?;
        let coalescing: Vec<String> = serde_json::from_str(&coalescing)?;
        let mut pending = Vec::new();
        for signal in self.signals(&run)? {
            if signal.id <= cursor || !filter.contains(&signal.kind) {
                continue;
            }
            let acked: bool = self.connection.query_row(
                "SELECT EXISTS(SELECT 1 FROM mailbox_ack WHERE consumer=?1 AND signal=?2)",
                params![consumer, signal.id],
                |row| row.get(0),
            )?;
            if !acked {
                pending.push(signal);
            }
        }
        Ok((pending, coalescing))
    }

    pub fn mailbox_next(&self, consumer: &str) -> Result<Option<Signal>> {
        let (pending, coalescing) = self.mailbox_pending(consumer)?;
        Ok(select_delivery(&pending, &coalescing).cloned())
    }

    pub fn mailbox_ack(&self, consumer: &str, signal_id: i64) -> Result<()> {
        let transaction = self.transaction()?;
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
        for signal in &pending {
            if signal.id == delivered.id
                || (coalescing.contains(&delivered.kind)
                    && same_coalescing_key(signal, delivered)
                    && signal.id < delivered.id)
            {
                transaction.execute(
                    "INSERT INTO mailbox_ack(consumer,signal) VALUES (?1,?2)",
                    params![consumer, signal.id],
                )?;
            }
        }
        let (remaining, _) = self.mailbox_pending(consumer)?;
        // A later coalesced delivery cannot advance past an earlier queue occurrence.
        let cursor = remaining
            .first()
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
        transaction.commit()?;
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

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    use tempfile::tempdir;

    #[test]
    fn pending_effect_is_uncertain_after_restart_and_never_replayed() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        db.create_run("task", "binding", &json!({"goal": "sample"}))?;
        let signal = db.next_signal("task")?.unwrap();
        let action = db
            .transition(
                "task",
                &json!({"step": 1}),
                signal.id,
                Some(&json!({"driver": "worker"})),
                "running",
            )?
            .unwrap();
        drop(db);
        let db = Journal::open(dir.path())?;
        assert!(db.recover("task")?);
        assert_eq!(db.run("task")?.status, "uncertain");
        assert_eq!(db.run("task")?.actions, 1);
        assert!(!db.recover("task")?);
        assert!(db
            .complete_action("task", action, "feedback", &json!("late"))
            .is_err());
        assert_eq!(db.signals("task")?.last().unwrap().kind, "stall");
        Ok(())
    }

    #[test]
    fn completed_action_survives_reopen_without_duplicate_delivery() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        db.create_run("task", "binding", &json!({}))?;
        let input = db.next_signal("task")?.unwrap();
        let action = db
            .transition("task", &json!(1), input.id, Some(&json!({})), "running")?
            .unwrap();
        db.complete_action("task", action, "feedback", &json!({"answer": 42}))?;
        assert!(db
            .complete_action("task", action, "feedback", &json!({}))
            .is_err());
        drop(db);
        let db = Journal::open(dir.path())?;
        assert!(!db.recover("task")?);
        let response = db.next_signal("task")?.unwrap();
        assert_eq!(response.action, Some(action));
        assert_eq!(response.payload["answer"], 42);
        assert_eq!(
            serde_json::from_slice::<Value>(
                &db.get_artifact(response.artifact.as_deref().unwrap())?
            )?,
            response.payload
        );
        db.transition("task", &json!(2), response.id, None, "complete")?;
        assert!(db.next_signal("task")?.is_none());
        assert!(db
            .transition("task", &json!(3), response.id, None, "complete")
            .is_err());
        assert_eq!(db.run("task")?.state, json!(2));
        Ok(())
    }

    #[test]
    fn failed_transition_is_atomic_and_cannot_ack_another_run() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        db.create_run("a", "a", &json!({}))?;
        db.create_run("b", "b", &json!({}))?;
        let input = db.next_signal("b")?.unwrap();
        assert!(db
            .transition(
                "a",
                &json!("changed"),
                input.id,
                Some(&json!({})),
                "running"
            )
            .is_err());
        assert_eq!(db.run("a")?.actions, 0);
        assert_eq!(db.run("a")?.state, Value::Null);
        assert!(db.next_signal("a")?.is_some());
        Ok(())
    }

    #[test]
    fn artifacts_are_content_addressed_and_corruption_is_detected() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        let hash = db.put_artifact(b"original")?;
        assert_eq!(db.put_artifact(b"original")?, hash);
        assert_eq!(db.get_artifact(&hash)?, b"original");
        assert!(db.get_artifact("../journal.sqlite").is_err());
        std::fs::write(dir.path().join("cas").join(&hash), b"tampered")?;
        assert!(db.get_artifact(&hash).is_err());
        assert!(db.put_artifact(b"original").is_err());
        Ok(())
    }

    #[test]
    fn knowledge_context_separates_authority_and_excludes_expiry() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        let source = db.put_artifact(b"evidence")?;
        db.add_knowledge(&json!({"id":"policy","kind":"decision","status":"stable","trust":"human","sources":[source],"text":"rule"}))?;
        db.add_knowledge(&json!({"id":"note","kind":"hypothesis","status":"draft","trust":"unverified","stale_after":20,"sources":[source],"text":"maybe"}))?;
        assert!(db.add_knowledge(&json!({"id":"bad","kind":"finding","status":"draft","trust":"unverified","sources":[]})).is_err());
        assert!(db.add_knowledge(&json!({"id":"bad-source","kind":"finding","status":"draft","trust":"unverified","stale_after":20,"sources":["missing"]})).is_err());
        let before = db.context(19)?;
        assert_eq!(before["confirmed"].as_array().unwrap().len(), 1);
        assert_eq!(before["provisional"].as_array().unwrap().len(), 1);
        assert_eq!(db.context(20)?["provisional"], json!([]));
        assert_eq!(db.context(20)?["confirmed"].as_array().unwrap().len(), 1);
        Ok(())
    }

    #[test]
    fn leases_fence_expired_holders_and_survive_reopen() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        let first = db.lease_acquire("workspace", "a", 10, 5)?;
        assert!(db.lease_acquire("workspace", "b", 14, 5).is_err());
        drop(db);
        let db = Journal::open(dir.path())?;
        let second = db.lease_acquire("workspace", "b", 15, 5)?;
        assert!(second > first);
        assert!(db.lease_release("workspace", "a", first).is_err());
        db.lease_release("workspace", "b", second)?;
        let third = db.lease_acquire("workspace", "a", 16, 5)?;
        assert!(third > second);
        assert!(db.lease_acquire("invalid", "a", 0, 0).is_err());
        Ok(())
    }
    #[test]
    fn mailbox_coalescing_preserves_interleaved_queue_and_consumer_independence() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        db.create_run("task", "binding", &json!({}))?;
        let first = db.append_signal("task", "timer", &json!({"target":"x","n":1}))?;
        let queue = db.append_signal("task", "human_steer", &json!({"text":"keep this"}))?;
        let latest = db.append_signal("task", "timer", &json!({"target":"x","n":2}))?;
        db.register_mailbox(
            "app",
            "task",
            &["timer".into(), "human_steer".into()],
            &["timer".into()],
        )?;
        db.register_mailbox(
            "audit",
            "task",
            &["timer".into(), "human_steer".into()],
            &[],
        )?;
        assert_eq!(db.mailbox_next("app")?.unwrap().id, latest);
        db.mailbox_ack("app", latest)?;
        drop(db);
        let db = Journal::open(dir.path())?;
        assert_eq!(db.mailbox_next("app")?.unwrap().id, queue);
        assert_eq!(db.mailbox_next("audit")?.unwrap().id, first);
        db.mailbox_ack("app", queue)?;
        assert!(db.mailbox_next("app")?.is_none());
        assert!(db.mailbox_ack("app", latest).is_err());
        assert!(db
            .register_mailbox("app", "task", &["other".into()], &[])
            .is_err());
        Ok(())
    }
    #[test]
    fn failed_cas_write_does_not_commit_action_completion() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        db.create_run("task", "binding", &json!({}))?;
        let input = db.next_signal("task")?.unwrap();
        let action = db
            .transition("task", &json!(1), input.id, Some(&json!({})), "running")?
            .unwrap();
        let payload = json!({"result": "important"});
        let hash = db.put_artifact(&serde_json::to_vec(&payload)?)?;
        std::fs::write(dir.path().join("cas").join(hash), b"corruption")?;
        assert!(db
            .complete_action("task", action, "feedback", &payload)
            .is_err());
        assert_eq!(db.signals("task")?.len(), 1);
        assert!(db.recover("task")?);
        Ok(())
    }
    #[test]
    fn coalescing_ack_keeps_notifications_arriving_during_delivery() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        db.create_run("task", "binding", &json!({}))?;
        db.register_mailbox("app", "task", &["timer".into()], &["timer".into()])?;
        let first = db.append_signal("task", "timer", &json!({"target":"x"}))?;
        assert_eq!(db.mailbox_next("app")?.unwrap().id, first);
        let newer = db.append_signal("task", "timer", &json!({"target":"x"}))?;
        db.mailbox_ack("app", first)?;
        assert_eq!(db.mailbox_next("app")?.unwrap().id, newer);
        Ok(())
    }

    #[test]
    fn schema_one_migration_preserves_existing_runs() -> Result<()> {
        let dir = tempdir()?;
        let db = Journal::open(dir.path())?;
        db.create_run("task", "binding", &json!({"goal":"retained"}))?;
        db.connection
            .execute_batch("ALTER TABLE actions DROP COLUMN artifact; PRAGMA user_version=1;")?;
        drop(db);
        let db = Journal::open(dir.path())?;
        assert_eq!(db.run("task")?.intent["goal"], "retained");
        let input = db.next_signal("task")?.unwrap();
        let action = db
            .transition("task", &json!(1), input.id, Some(&json!({})), "running")?
            .unwrap();
        db.complete_action("task", action, "feedback", &json!({"ok":true}))?;
        assert!(db.next_signal("task")?.unwrap().artifact.is_some());
        Ok(())
    }
    #[test]
    fn concurrent_transitions_cannot_dispatch_the_same_input_twice() -> Result<()> {
        let dir = tempdir()?;
        let first = Journal::open(dir.path())?;
        first.create_run("task", "binding", &json!({}))?;
        let signal = first.next_signal("task")?.unwrap().id;
        let second = Journal::open(dir.path())?;
        let barrier = std::sync::Arc::new(std::sync::Barrier::new(2));
        let other_barrier = barrier.clone();
        let a = std::thread::spawn(move || {
            barrier.wait();
            first.transition(
                "task",
                &json!(1),
                signal,
                Some(&json!({"which":"a"})),
                "running",
            )
        });
        let b = std::thread::spawn(move || {
            other_barrier.wait();
            second.transition(
                "task",
                &json!(2),
                signal,
                Some(&json!({"which":"b"})),
                "running",
            )
        });
        let results = [a.join().unwrap(), b.join().unwrap()];
        assert_eq!(results.iter().filter(|result| result.is_ok()).count(), 1);
        let db = Journal::open(dir.path())?;
        assert_eq!(db.run("task")?.actions, 1);
        assert!(db.recover("task")?);
        Ok(())
    }
}
