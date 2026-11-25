//! Consumer-owned persistence and resource-ownership contracts, independent of storage engines.
use anyhow::Result;
use serde::Serialize;
use serde_json::Value;

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

pub trait RunStore {
    /// Create a new run or reopen its exact binding and Intent, never reset it.
    fn open_run(&self, id: &str, binding: &str, intent: &Value) -> Result<Run>;
    fn run(&self, id: &str) -> Result<Run>;
    fn recover(&self, id: &str) -> Result<bool>;
    fn complete_action(&self, id: &str, action: i64, kind: &str, payload: &Value) -> Result<()>;
}

pub trait Mailbox {
    fn receive(&self, id: &str) -> Result<Option<Signal>>;
    /// Commit next application state and acknowledge its delivered input together.
    fn commit(&self, id: &str, state: &Value, signal: i64, status: &str) -> Result<()>;
}

pub struct Reservation<'a> {
    pub run: &'a str,
    pub state: &'a Value,
    pub signal: i64,
    pub effect: &'a Value,
    pub resource: &'a str,
    pub now: i64,
    pub ttl: i64,
}
#[derive(Debug)]
pub struct ReservationReceipt {
    pub action: i64,
    pub token: u64,
}

pub trait LeaseManager {
    /// Busy returns None with the mailbox input, application state and action count unchanged.
    fn reserve(&self, request: Reservation<'_>) -> Result<Option<ReservationReceipt>>;
}
pub trait ArtifactStore {
    fn get_artifact(&self, hash: &str) -> Result<Vec<u8>>;
}
pub trait KnowledgeStore {
    fn context(&self, now: i64) -> Result<Value>;
}

/// Persisted invocation evidence for operator inspection, not an inferred external result.
#[derive(Debug, Serialize)]
pub struct Action {
    pub id: i64,
    pub run: String,
    pub generation: i64,
    pub status: String,
    pub effect: Value,
    pub resource: Option<String>,
    pub lease_token: Option<u64>,
    pub artifact: Option<String>,
}
