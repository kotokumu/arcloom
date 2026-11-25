//! SQLite WAL and local CAS implementation of Agent OS storage contracts.
use crate::storage::ArtifactStore;
pub use crate::storage::{Run, Signal};
use anyhow::{ensure, Result};
pub use artifacts::LocalCas;
use rusqlite::{Connection, Transaction, TransactionBehavior};
use std::{fs, path::Path, rc::Rc, time::Duration};

mod artifacts;
mod knowledge;
mod leases;
mod mailbox;
mod runs;
#[cfg(test)]
mod tests;

/// Evidence storage required by committed action outputs and Knowledge provenance.
/// A successful write must preserve immutable bytes for later reads before returning.
/// Journal verifies the content address through this same port before publishing it.
pub trait EvidenceStore: ArtifactStore {
    fn put_artifact(&self, bytes: &[u8]) -> Result<String>;
}

pub struct Journal {
    connection: Connection,
    artifacts: Rc<dyn EvidenceStore>,
}
impl Journal {
    pub fn open(dir: &Path) -> Result<Self> {
        Self::open_with_artifacts(dir, Rc::new(LocalCas::open(&dir.join("cas"))?))
    }

    pub fn open_with_artifacts(dir: &Path, artifacts: Rc<dyn EvidenceStore>) -> Result<Self> {
        fs::create_dir_all(dir)?;
        let connection = Connection::open(dir.join("journal.sqlite"))?;
        connection.busy_timeout(Duration::from_secs(5))?;
        connection.pragma_update(None, "foreign_keys", "ON")?;
        connection.pragma_update(None, "journal_mode", "WAL")?;
        connection.pragma_update(None, "synchronous", "FULL")?;
        let transaction = Transaction::new_unchecked(&connection, TransactionBehavior::Immediate)?;
        let version: i64 =
            transaction.pragma_query_value(None, "user_version", |row| row.get(0))?;
        ensure!(
            (0..=3).contains(&version),
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
        if version < 3 {
            transaction.execute_batch("ALTER TABLE actions ADD COLUMN resource TEXT; ALTER TABLE actions ADD COLUMN lease_token INTEGER; PRAGMA user_version=3;")?;
        }
        transaction.commit()?;
        Ok(Self {
            connection,
            artifacts,
        })
    }

    fn transaction(&self) -> Result<Transaction<'_>> {
        Ok(Transaction::new_unchecked(
            &self.connection,
            TransactionBehavior::Immediate,
        )?)
    }
}
