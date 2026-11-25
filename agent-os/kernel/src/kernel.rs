//! Advance one durable mailbox delivery. Only the outer host waits on a provider.
use crate::{
    ports::{
        Application, Dispatcher, Effect, Invocation, LeaseFence, Pending, ReadError, Syscalls,
    },
    storage::{ArtifactStore, KnowledgeStore, LeaseManager, Mailbox, Reservation, Run, RunStore},
};
use anyhow::{Context, Result};
use serde_json::{json, Value};
use std::{collections::BTreeSet, rc::Rc};

pub struct Kernel<'a> {
    pub runs: &'a dyn RunStore,
    pub inbox: &'a dyn Mailbox,
    pub leases: &'a dyn LeaseManager,
    pub artifacts: Rc<dyn ArtifactStore>,
    pub knowledge: Rc<dyn KnowledgeStore>,
    pub dispatch: &'a dyn Dispatcher,
}

pub struct RunPolicy<'a> {
    pub grants: &'a [String],
    pub max_actions: u64,
    pub lease_ttl_seconds: i64,
}

struct GuestAccess {
    capabilities: BTreeSet<String>,
    can_read_knowledge: bool,
    can_read_artifact: bool,
    knowledge: Rc<dyn KnowledgeStore>,
    artifacts: Rc<dyn ArtifactStore>,
    artifact: Option<String>,
    now: i64,
}

impl Syscalls for GuestAccess {
    fn permitted(&self, capability: &str) -> bool {
        self.capabilities.contains(capability)
    }
    fn knowledge(&self) -> Result<Value, ReadError> {
        if !self.can_read_knowledge {
            return Err(ReadError::Denied);
        }
        self.knowledge
            .context(self.now)
            .map_err(|_| ReadError::Unavailable)
    }
    fn artifact(&self, hash: &str) -> Result<Vec<u8>, ReadError> {
        if !self.can_read_artifact || self.artifact.as_deref() != Some(hash) {
            return Err(ReadError::Denied);
        }
        self.artifacts
            .get_artifact(hash)
            .map_err(|_| ReadError::Unavailable)
    }
}

pub enum Step {
    Idle(Run),
    /// Reservation unavailable. Nothing was consumed; the same input remains eligible.
    Busy {
        resource: String,
    },
    Submitted(Submission),
}

/// The ticket carries the immutable receipt from reservation to completion.
pub struct Submission {
    run: String,
    action: i64,
    capability: String,
    pub pending: Box<dyn Pending>,
}

impl Submission {
    pub fn into_parts(self) -> (String, i64, String, Box<dyn Pending>) {
        (self.run, self.action, self.capability, self.pending)
    }
}

impl Kernel<'_> {
    pub fn step(
        &self,
        id: &str,
        policy: &RunPolicy<'_>,
        now: i64,
        app: &mut dyn Application,
    ) -> Result<Step> {
        let current = self.runs.run(id)?;
        if matches!(
            current.status.as_str(),
            "running" | "completed" | "uncertain" | "budget_hit" | "denied" | "failed"
        ) {
            return Ok(Step::Idle(current));
        }
        let Some(signal) = self.inbox.receive(id)? else {
            return Ok(Step::Idle(current));
        };
        let access = Rc::new(GuestAccess {
            capabilities: policy
                .grants
                .iter()
                .filter(|c| self.dispatch.authorized(c))
                .cloned()
                .collect(),
            can_read_knowledge: policy.grants.iter().any(|c| c == "knowledge.read"),
            can_read_artifact: policy.grants.iter().any(|c| c == "artifact.read"),
            knowledge: self.knowledge.clone(),
            artifacts: self.artifacts.clone(),
            artifact: signal.artifact.clone(),
            now,
        });
        let next = app.on_signal(&current.state, &signal, access.clone())?;
        let terminal = |state: &Value, status| {
            self.inbox.commit(id, state, signal.id, status)?;
            Ok(Step::Idle(self.runs.run(id)?))
        };
        let (capability, input) = match next.effect {
            Effect::Wait => return terminal(&next.state, "waiting"),
            Effect::Complete => return terminal(&next.state, "completed"),
            Effect::Denied => return terminal(&current.state, "denied"),
            Effect::Invoke { capability, input } => (capability, input),
        };
        if !access.permitted(&capability) {
            return terminal(&current.state, "denied");
        }
        if current.actions >= policy.max_actions {
            return terminal(&current.state, "budget_hit");
        }
        let provider = self
            .dispatch
            .provider(&capability)
            .context("unresolved capability")?;
        let resource = self
            .dispatch
            .resource(&capability)
            .context("missing resource binding")?
            .to_owned();
        let effect = json!({"capability":capability, "provider":provider, "input":input});
        let Some(receipt) = self.leases.reserve(Reservation {
            run: id,
            state: &next.state,
            signal: signal.id,
            effect: &effect,
            resource: &resource,
            now,
            ttl: policy.lease_ttl_seconds,
        })?
        else {
            return Ok(Step::Busy { resource });
        };
        let action = receipt.action;
        let request = Invocation {
            run: id.to_owned(),
            action,
            generation: action,
            input,
            intent: current.intent,
            lease: LeaseFence {
                resource,
                token: receipt.token,
            },
        };
        match self.dispatch.submit(&capability, request) {
            Ok(pending) => Ok(Step::Submitted(Submission {
                run: id.to_owned(),
                action,
                capability,
                pending,
            })),
            Err(_) => {
                self.runs.recover(id)?;
                Ok(Step::Idle(self.runs.run(id)?))
            }
        }
    }
}
