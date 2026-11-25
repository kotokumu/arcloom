//! Component adapter: the guest owns business state and transition decisions.
use crate::model::{Limits, Transition};
use anyhow::{ensure, Result};
use serde_json::Value;
use wasmtime::{
    component::{Component, Linker},
    Config, Engine, Store, StoreLimits, StoreLimitsBuilder,
};

wasmtime::component::bindgen!({ path: "wit", world: "business-app" });

pub trait Application {
    fn on_signal(&mut self, state: &Value, signal: &Value) -> Result<Transition>;
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
impl Application for WasmApplication {
    fn on_signal(&mut self, state: &Value, signal: &Value) -> Result<Transition> {
        let state = serde_json::to_string(state)?;
        let signal = serde_json::to_string(signal)?;
        ensure!(
            state.len().saturating_add(signal.len()) <= self.limits.memory_bytes,
            "component input exceeds memory budget"
        );
        let limits = guest_limits(self.limits.memory_bytes);
        let mut store = Store::new(&self.engine, limits);
        store.limiter(|s| s);
        store.set_fuel(self.limits.fuel)?;
        // Empty linker: no ambient WASI filesystem, network, environment or clocks.
        let linker = Linker::new(&self.engine);
        let app = BusinessApp::instantiate(&mut store, &self.component, &linker)?;
        let output = app.call_on_signal(&mut store, &state, &signal)?;
        ensure!(
            output.len() <= self.limits.memory_bytes,
            "component result exceeds memory budget"
        );
        Ok(serde_json::from_str(&output)?)
    }
}

// This prototype ABI admits one guest memory/table, bounding aggregate guest
// memory rather than granting the configured budget separately to many memories.
fn guest_limits(bytes: usize) -> StoreLimits {
    StoreLimitsBuilder::new()
        .memory_size(bytes)
        .memories(1)
        .tables(1)
        .table_elements((bytes / std::mem::size_of::<usize>()).min(u32::MAX as usize))
        .instances(8)
        .build()
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn component_cannot_multiply_its_memory_budget() -> Result<()> {
        let mut config = Config::new();
        config.wasm_component_model(true);
        let engine = Engine::new(&config)?;
        let bytes = wat::parse_str(
            r#"(component
            (core module $m (memory 1))
            (core instance (instantiate $m))
            (core instance (instantiate $m)))"#,
        )?;
        let component = Component::new(&engine, bytes)?;
        let mut store = Store::new(&engine, guest_limits(65536));
        store.limiter(|s| s);
        let error = Linker::<StoreLimits>::new(&engine)
            .instantiate(&mut store, &component)
            .expect_err("two memories must be rejected");
        assert!(format!("{error:#}").contains("memory"), "{error:#}");
        Ok(())
    }
}
