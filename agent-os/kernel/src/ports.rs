//! Contracts consumed by a kernel turn. Transport and persistence adapters are
//! constructed by the host; business documents remain owned by the guest.
use crate::storage::Signal;
use anyhow::Result;
use serde_json::Value;

pub enum Effect {
    Wait,
    Complete,
    Denied,
    Invoke { capability: String, input: Value },
}

pub struct Transition {
    pub state: Value,
    pub effect: Effect,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ReadError {
    Denied,
    Unavailable,
}

pub trait Syscalls {
    fn permitted(&self, capability: &str) -> bool;
    fn knowledge(&self) -> Result<Value, ReadError>;
    fn artifact(&self, hash: &str) -> Result<Vec<u8>, ReadError>;
}

pub trait Application {
    fn on_signal(
        &mut self,
        state: &Value,
        signal: &Signal,
        host: std::rc::Rc<dyn Syscalls>,
    ) -> Result<Transition>;
}

#[derive(serde::Serialize, Debug, Clone)]
pub struct LeaseFence {
    pub resource: String,
    pub token: u64,
}

#[derive(serde::Serialize, Debug, Clone)]
pub struct Invocation {
    pub run: String,
    pub action: i64,
    pub generation: i64,
    pub input: Value,
    pub intent: Value,
    pub lease: LeaseFence,
}

pub trait Dispatcher {
    fn authorized(&self, capability: &str) -> bool;
    fn resource(&self, capability: &str) -> Option<&str>;
    fn provider(&self, capability: &str) -> Option<&str>;
    fn submit(&self, capability: &str, request: Invocation) -> Result<Box<dyn Pending>>;
}

/// The host decides where and when to await provider work.
pub trait Pending {
    fn finish(self: Box<Self>) -> Result<Value>;
}
