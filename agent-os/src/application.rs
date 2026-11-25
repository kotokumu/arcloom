//! Typed Component adapter; callbacks stage effects and cannot dispatch them.
use crate::model::Limits;
use crate::ports::{Application, Effect, ReadError, Syscalls, Transition as KernelTransition};
use crate::storage::Signal as KernelSignal;
use anyhow::{anyhow, ensure, Result};
use serde_json::Value;
use wasmtime::{
    component::{Component, HasData, Linker, Resource, ResourceTable},
    Config, Engine, Store, StoreLimits, StoreLimitsBuilder,
};

pub struct LeaseReservation {
    capability: String,
}
pub struct DispatchTicket {
    capability: String,
    input: Value,
}
wasmtime::component::bindgen!({
    path: "wit", world: "business-app",
    with: {
        "arcloom:os/syscalls/lease": LeaseReservation,
        "arcloom:os/syscalls/ticket": DispatchTicket,
    }
});
use arcloom::os::syscalls::{Host, HostLease, HostTicket, Outcome, SyscallError};

struct Invocation {
    limits: StoreLimits,
    table: ResourceTable,
    host: std::rc::Rc<dyn Syscalls>,
    max_bytes: usize,
    charged: usize,
}
struct InvocationHost<'a> {
    table: &'a mut ResourceTable,
    host: &'a dyn Syscalls,
    max_bytes: usize,
    charged: &'a mut usize,
}
struct InvocationData;
impl HasData for InvocationData {
    type Data<'a> = InvocationHost<'a>;
}
impl InvocationHost<'_> {
    fn charge(&mut self, bytes: usize) -> Result<(), SyscallError> {
        *self.charged = self
            .charged
            .checked_add(bytes)
            .ok_or(SyscallError::LimitExceeded)?;
        if *self.charged > self.max_bytes {
            return Err(SyscallError::LimitExceeded);
        }
        Ok(())
    }
}
impl Invocation {
    fn view(&mut self) -> InvocationHost<'_> {
        InvocationHost {
            table: &mut self.table,
            host: self.host.as_ref(),
            max_bytes: self.max_bytes,
            charged: &mut self.charged,
        }
    }
    fn finish(
        &mut self,
        transition: arcloom::os::syscalls::Transition,
    ) -> Result<KernelTransition> {
        ensure!(
            transition.state.len() <= self.max_bytes,
            "application state exceeds limit"
        );
        let state = serde_json::from_str(&transition.state)?;
        let effect = match transition.effect {
            Outcome::Wait => Effect::Wait,
            Outcome::Complete => Effect::Complete,
            Outcome::Denied => Effect::Denied,
            Outcome::Dispatch(handle) => {
                let ticket = self.table.delete(handle)?;
                ensure!(
                    self.host.permitted(&ticket.capability),
                    "dispatch permission changed"
                );
                Effect::Invoke {
                    capability: ticket.capability,
                    input: ticket.input,
                }
            }
        };
        Ok(KernelTransition { state, effect })
    }
}
impl HostLease for InvocationHost<'_> {
    fn drop(&mut self, handle: Resource<LeaseReservation>) -> Result<()> {
        self.table.delete(handle)?;
        Ok(())
    }
}
impl HostTicket for InvocationHost<'_> {
    fn drop(&mut self, handle: Resource<DispatchTicket>) -> Result<()> {
        self.table.delete(handle)?;
        Ok(())
    }
}
impl Host for InvocationHost<'_> {
    fn lease_acquire(
        &mut self,
        capability: String,
    ) -> Result<Resource<LeaseReservation>, SyscallError> {
        if !self.host.permitted(&capability) {
            return Err(SyscallError::Denied);
        }
        self.charge(
            capability
                .len()
                .saturating_mul(2)
                .saturating_add(std::mem::size_of::<LeaseReservation>()),
        )?;
        self.table
            .push(LeaseReservation { capability })
            .map_err(|_| SyscallError::LimitExceeded)
    }
    fn dispatch(
        &mut self,
        reservation: Resource<LeaseReservation>,
        input: String,
    ) -> Result<Resource<DispatchTicket>, SyscallError> {
        let lease = self
            .table
            .get(&reservation)
            .map_err(|_| SyscallError::InvalidHandle)?;
        if !self.host.permitted(&lease.capability) {
            return Err(SyscallError::Denied);
        }
        let capability_bytes = lease.capability.len();
        self.charge(
            input
                .len()
                .saturating_add(capability_bytes)
                .saturating_add(std::mem::size_of::<DispatchTicket>()),
        )?;
        let input = serde_json::from_str(&input).map_err(|_| SyscallError::InvalidInput)?;
        let ticket = DispatchTicket {
            capability: self
                .table
                .get(&reservation)
                .map_err(|_| SyscallError::InvalidHandle)?
                .capability
                .clone(),
            input,
        };
        self.table
            .push(ticket)
            .map_err(|_| SyscallError::LimitExceeded)
    }
    fn knowledge_query(&mut self) -> Result<arcloom::os::syscalls::ContextView, SyscallError> {
        let result = self.host.knowledge().map_err(read_error)?;
        self.charge(result.to_string().len())?;
        context_view(&result).map_err(|_| SyscallError::Unavailable)
    }
    fn artifact_get(&mut self, hash: String) -> Result<Vec<u8>, SyscallError> {
        let result = self.host.artifact(&hash).map_err(read_error)?;
        self.charge(result.len())?;
        Ok(result)
    }
}

fn read_error(error: ReadError) -> SyscallError {
    match error {
        ReadError::Denied => SyscallError::Denied,
        ReadError::Unavailable => SyscallError::Unavailable,
    }
}

fn context_view(value: &Value) -> Result<arcloom::os::syscalls::ContextView> {
    use arcloom::os::syscalls::{
        ContextView, KnowledgeKind, KnowledgeNode, KnowledgeStatus, KnowledgeTrust,
    };
    fn nodes(value: &Value) -> Result<Vec<KnowledgeNode>> {
        value
            .as_array()
            .ok_or_else(|| anyhow!("context partition must be array"))?
            .iter()
            .map(|node| {
                let text = |key: &str| {
                    node[key]
                        .as_str()
                        .ok_or_else(|| anyhow!("invalid knowledge {key}"))
                };
                let kind = match text("kind")? {
                    "decision" => KnowledgeKind::Decision,
                    "finding" => KnowledgeKind::Finding,
                    "hypothesis" => KnowledgeKind::Hypothesis,
                    "disproved" => KnowledgeKind::Disproved,
                    _ => return Err(anyhow!("invalid kind")),
                };
                let status = match text("status")? {
                    "draft" => KnowledgeStatus::Draft,
                    "stable" => KnowledgeStatus::Stable,
                    "deprecated" => KnowledgeStatus::Deprecated,
                    _ => return Err(anyhow!("invalid status")),
                };
                let trust = match text("trust")? {
                    "unverified" => KnowledgeTrust::Unverified,
                    "machine" => KnowledgeTrust::Machine,
                    "human" => KnowledgeTrust::Human,
                    _ => return Err(anyhow!("invalid trust")),
                };
                let sources = node["sources"]
                    .as_array()
                    .ok_or_else(|| anyhow!("invalid sources"))?
                    .iter()
                    .map(|s| {
                        s.as_str()
                            .or_else(|| s["sha256"].as_str())
                            .map(str::to_owned)
                            .ok_or_else(|| anyhow!("invalid source"))
                    })
                    .collect::<Result<Vec<_>>>()?;
                let mut extension = node
                    .as_object()
                    .ok_or_else(|| anyhow!("invalid node"))?
                    .clone();
                for key in ["id", "kind", "status", "trust", "stale_after", "sources"] {
                    extension.remove(key);
                }
                Ok(KnowledgeNode {
                    id: text("id")?.to_owned(),
                    kind,
                    status,
                    trust,
                    stale_after: node["stale_after"].as_i64(),
                    sources,
                    payload: Value::Object(extension).to_string(),
                })
            })
            .collect()
    }
    Ok(ContextView {
        confirmed: nodes(&value["confirmed"])?,
        provisional: nodes(&value["provisional"])?,
    })
}

pub struct WasmApplication {
    engine: Engine,
    component: Component,
    limits: Limits,
}
impl WasmApplication {
    pub fn new(bytes: &[u8], limits: &Limits) -> Result<Self> {
        let mut config = Config::new();
        config.wasm_component_model(true).consume_fuel(true);
        let engine = Engine::new(&config)?;
        let component = Component::new(&engine, bytes)?;
        Ok(Self {
            engine,
            component,
            limits: limits.clone(),
        })
    }
}
impl WasmApplication {
    fn invoke(
        &self,
        state: String,
        signal: arcloom::os::syscalls::Signal,
        host: std::rc::Rc<dyn Syscalls>,
    ) -> Result<KernelTransition> {
        let mut store = Store::new(
            &self.engine,
            Invocation {
                limits: guest_limits(self.limits.memory_bytes),
                table: ResourceTable::new(),
                host,
                max_bytes: self.limits.memory_bytes,
                charged: 0,
            },
        );
        store.limiter(|s| &mut s.limits);
        store.set_fuel(self.limits.fuel)?;
        let mut linker = Linker::new(&self.engine);
        BusinessApp::add_to_linker::<_, InvocationData>(&mut linker, |s: &mut Invocation| {
            s.view()
        })?;
        let app = BusinessApp::instantiate(&mut store, &self.component, &linker)?;
        let result = app
            .call_on_signal(&mut store, &state, &signal)?
            .map_err(|s| anyhow!("application rejected signal: {s}"))?;
        store.data_mut().finish(result)
    }
}
impl Application for WasmApplication {
    fn on_signal(
        &mut self,
        state: &Value,
        signal: &KernelSignal,
        host: std::rc::Rc<dyn Syscalls>,
    ) -> Result<KernelTransition> {
        let state = state.to_string();
        let signal = arcloom::os::syscalls::Signal {
            id: signal.id,
            kind: signal.kind.clone(),
            payload: signal.payload.to_string(),
            action: signal.action,
            generation: signal.generation,
            artifact: signal.artifact.clone(),
        };
        ensure!(
            state.len().saturating_add(signal.payload.len()) <= self.limits.memory_bytes,
            "component input exceeds memory budget"
        );
        self.invoke(state, signal, host)
    }
}

fn guest_limits(bytes: usize) -> StoreLimits {
    StoreLimitsBuilder::new()
        .memory_size(bytes)
        .memories(1)
        .tables(2)
        // The Rust guest and canonical resource adapter each need one table.
        // Divide the table-element budget, so two tables cannot double it.
        .table_elements((bytes / 2 / std::mem::size_of::<usize>()).min(u32::MAX as usize))
        .instances(8)
        .build()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{cell::Cell, rc::Rc};
    #[derive(Clone)]
    struct HostState {
        allowed: bool,
        reads: Rc<Cell<u32>>,
    }
    impl Syscalls for HostState {
        fn permitted(&self, cap: &str) -> bool {
            self.allowed && cap == "example.action"
        }
        fn knowledge(&self) -> std::result::Result<Value, ReadError> {
            self.reads.set(self.reads.get() + 1);
            if !self.allowed {
                return Err(ReadError::Denied);
            }
            Ok(serde_json::json!({"confirmed":[],"provisional":[]}))
        }
        fn artifact(&self, hash: &str) -> std::result::Result<Vec<u8>, ReadError> {
            if !self.allowed || hash != "own-signal" {
                return Err(ReadError::Denied);
            }
            Ok(b"own bytes".to_vec())
        }
    }
    fn invocation(host: &HostState) -> Invocation {
        Invocation {
            limits: guest_limits(65536),
            table: ResourceTable::new(),
            host: Rc::new(host.clone()),
            max_bytes: 65536,
            charged: 0,
        }
    }
    #[test]
    fn denied_and_forged_reservations_cannot_stage_dispatch() {
        let denied = HostState {
            allowed: false,
            reads: Rc::new(Cell::new(0)),
        };
        assert!(matches!(
            invocation(&denied)
                .view()
                .lease_acquire("example.action".into()),
            Err(SyscallError::Denied)
        ));
        let allowed = HostState {
            allowed: true,
            reads: Rc::new(Cell::new(0)),
        };
        assert!(matches!(
            invocation(&allowed)
                .view()
                .dispatch(Resource::new_borrow(123), "{}".into()),
            Err(SyscallError::InvalidHandle)
        ));
    }
    #[test]
    fn ticket_is_callback_local_bound_to_capability_and_consumed_once() -> Result<()> {
        let host = HostState {
            allowed: true,
            reads: Rc::new(Cell::new(0)),
        };
        let mut call = invocation(&host);
        let lease = call
            .view()
            .lease_acquire("example.action".into())
            .map_err(|_| anyhow!("lease"))?;
        let ticket = call
            .view()
            .dispatch(Resource::new_borrow(lease.rep()), "{\"x\":1}".into())
            .map_err(|_| anyhow!("dispatch"))?;
        let rep = ticket.rep();
        let result = call.finish(arcloom::os::syscalls::Transition {
            state: "null".into(),
            effect: Outcome::Dispatch(ticket),
        })?;
        match result.effect {
            Effect::Invoke { capability, input } => {
                assert_eq!(capability, "example.action");
                assert_eq!(input["x"], 1);
            }
            _ => panic!("expected invocation"),
        }
        assert!(call
            .finish(arcloom::os::syscalls::Transition {
                state: "null".into(),
                effect: Outcome::Dispatch(Resource::new_own(rep))
            })
            .is_err());
        assert_eq!(host.reads.get(), 0, "context must be demand-driven");
        Ok(())
    }
    #[test]
    fn invalid_state_does_not_commit_ticket_and_reads_enforce_host_scope() -> Result<()> {
        let host = HostState {
            allowed: true,
            reads: Rc::new(Cell::new(0)),
        };
        let mut call = invocation(&host);
        let lease = call
            .view()
            .lease_acquire("example.action".into())
            .map_err(|_| anyhow!("lease"))?;
        let ticket = call
            .view()
            .dispatch(Resource::new_borrow(lease.rep()), "{}".into())
            .map_err(|_| anyhow!("dispatch"))?;
        assert!(call
            .finish(arcloom::os::syscalls::Transition {
                state: "invalid json".into(),
                effect: Outcome::Dispatch(ticket)
            })
            .is_err());
        assert!(matches!(
            call.view().artifact_get("another-run".into()),
            Err(SyscallError::Denied)
        ));
        assert_eq!(
            call.view()
                .artifact_get("own-signal".into())
                .map_err(|_| anyhow!("read"))?,
            b"own bytes"
        );
        assert!(call.view().knowledge_query().is_ok());
        assert_eq!(host.reads.get(), 1);
        Ok(())
    }
    #[test]
    fn discarded_tickets_do_not_become_effects_and_handle_types_cannot_be_forged() -> Result<()> {
        let host = HostState {
            allowed: true,
            reads: Rc::new(Cell::new(0)),
        };
        let mut call = invocation(&host);
        let lease = call
            .view()
            .lease_acquire("example.action".into())
            .map_err(|_| anyhow!("lease"))?;
        let ticket = call
            .view()
            .dispatch(Resource::new_borrow(lease.rep()), "{}".into())
            .map_err(|_| anyhow!("dispatch"))?;
        assert!(matches!(
            call.view()
                .dispatch(Resource::new_borrow(ticket.rep()), "{}".into()),
            Err(SyscallError::InvalidHandle)
        ));
        let transition = call.finish(arcloom::os::syscalls::Transition {
            state: "null".into(),
            effect: Outcome::Complete,
        })?;
        assert!(matches!(transition.effect, Effect::Complete));
        Ok(())
    }
    #[test]
    fn typed_knowledge_keeps_metadata_out_of_opaque_extension() -> Result<()> {
        let input = serde_json::json!({"confirmed":[{"id":"rule","kind":"decision","status":"stable","trust":"human","sources":[{"sha256":"source"}],"text":"rule body"}],"provisional":[]});
        let view = context_view(&input)?;
        assert_eq!(view.confirmed[0].id, "rule");
        assert_eq!(view.confirmed[0].sources, vec!["source"]);
        assert_eq!(
            serde_json::from_str::<Value>(&view.confirmed[0].payload)?,
            serde_json::json!({"text":"rule body"})
        );
        Ok(())
    }
    #[test]
    fn repeated_dispatch_charges_each_copied_capability() -> Result<()> {
        let host = HostState {
            allowed: true,
            reads: Rc::new(Cell::new(0)),
        };
        let mut call = invocation(&host);
        let cap = "example.action";
        call.max_bytes = cap.len() * 2
            + std::mem::size_of::<LeaseReservation>()
            + 2 * (2 + std::mem::size_of::<DispatchTicket>());
        let lease = call
            .view()
            .lease_acquire(cap.into())
            .map_err(|_| anyhow!("lease"))?;
        assert!(call
            .view()
            .dispatch(Resource::new_borrow(lease.rep()), "{}".into())
            .is_ok());
        assert!(matches!(
            call.view()
                .dispatch(Resource::new_borrow(lease.rep()), "{}".into()),
            Err(SyscallError::LimitExceeded)
        ));
        Ok(())
    }
    #[test]
    fn component_adapter_tables_fit_but_a_third_table_is_rejected() -> Result<()> {
        let mut config = Config::new();
        config.wasm_component_model(true);
        let engine = Engine::new(&config)?;
        let two = wat::parse_str(
            r#"(component (core module $m (table 1 funcref)) (core instance (instantiate $m)) (core instance (instantiate $m)))"#,
        )?;
        let three = wat::parse_str(
            r#"(component (core module $m (table 1 funcref)) (core instance (instantiate $m)) (core instance (instantiate $m)) (core instance (instantiate $m)))"#,
        )?;
        let mut store = Store::new(&engine, guest_limits(65536));
        store.limiter(|s| s);
        Linker::<StoreLimits>::new(&engine)
            .instantiate(&mut store, &Component::new(&engine, two)?)?;
        let mut store = Store::new(&engine, guest_limits(65536));
        store.limiter(|s| s);
        let error = Linker::<StoreLimits>::new(&engine)
            .instantiate(&mut store, &Component::new(&engine, three)?)
            .expect_err("third table rejected");
        assert!(format!("{error:#}").contains("table"));
        Ok(())
    }
    #[test]
    fn component_cannot_multiply_its_memory_budget() -> Result<()> {
        let mut config = Config::new();
        config.wasm_component_model(true);
        let engine = Engine::new(&config)?;
        let bytes = wat::parse_str(
            r#"(component (core module $m (memory 1)) (core instance (instantiate $m)) (core instance (instantiate $m)))"#,
        )?;
        let component = Component::new(&engine, bytes)?;
        let mut store = Store::new(&engine, guest_limits(65536));
        store.limiter(|s| s);
        let error = Linker::<StoreLimits>::new(&engine)
            .instantiate(&mut store, &component)
            .expect_err("two memories must be rejected");
        assert!(format!("{error:#}").contains("memory"));
        Ok(())
    }
}
