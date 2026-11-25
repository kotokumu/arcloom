//! Composition of declared capability dependencies, independent of kernel policy.
use crate::{
    drivers::CommandDriver,
    model::Manifest,
    ports::{Dispatcher, Invocation, Pending},
};
use anyhow::{ensure, Context, Result};
use serde::Deserialize;
use serde_json::Value;
use std::{
    collections::{BTreeMap, BTreeSet},
    path::{Path, PathBuf},
    sync::Arc,
    thread::JoinHandle,
};

/// Minimal provider contract owned by the capability registry. Implementations
/// may use native code, subprocesses or other host-selected transports.
pub trait Driver: Send + Sync {
    fn invoke(&self, request: &Value) -> Result<Value>;
}

/// A provider receives only the handles named in its own `requires` declaration.
#[derive(Clone)]
pub struct Dependencies(BTreeMap<String, CapabilityHandle>);
impl Dependencies {
    pub fn get(&self, capability: &str) -> Result<CapabilityHandle> {
        self.0
            .get(capability)
            .cloned()
            .context("EPERM: dependency was not declared")
    }
}

#[derive(Clone)]
pub struct CapabilityHandle {
    capability: String,
    provider: String,
    resource: String,
    granted: bool,
    driver: Arc<dyn Driver>,
}
impl CapabilityHandle {
    pub fn capability(&self) -> &str {
        &self.capability
    }
    pub fn provider(&self) -> &str {
        &self.provider
    }
    pub fn invoke(&self, request: &Value) -> Result<Value> {
        ensure!(self.granted, "EPERM: dependency capability is not granted");
        ensure!(
            request["lease"]["resource"].as_str() == Some(self.resource.as_str())
                && request["lease"]["token"].as_u64().is_some_and(|v| v > 0),
            "dependency invocation does not carry the shared resource fence"
        );
        let mut request = request.clone();
        let object = request
            .as_object_mut()
            .context("driver request must be an object")?;
        // Provider identity and invoked capability come from the binding, not
        // from caller-controlled data, even for nested provider invocations.
        object.insert("capability".into(), Value::String(self.capability.clone()));
        object.insert("provider".into(), Value::String(self.provider.clone()));
        self.driver.invoke(&request)
    }
}

/// Host-selected adapters implement this factory; kernel contracts do not change
/// when a new transport is registered in the composition root.
pub trait AdapterFactory: Send + Sync {
    fn build(&self, config: &Value, dependencies: Dependencies) -> Result<Arc<dyn Driver>>;
}
pub type Factories = BTreeMap<String, Arc<dyn AdapterFactory>>;

pub struct Registry {
    handles: BTreeMap<String, CapabilityHandle>,
    roots: BTreeSet<String>,
}
impl Registry {
    pub fn from_manifest(manifest: &Manifest) -> Result<Self> {
        Self::with_factories(manifest, Self::default_factories(manifest)?)
    }
    pub fn default_factories(manifest: &Manifest) -> Result<Factories> {
        let base = if manifest.base_dir.as_os_str().is_empty() {
            std::env::current_dir()?
        } else {
            manifest.base_dir.clone()
        };
        let mut factories: Factories = BTreeMap::new();
        factories.insert("command".into(), Arc::new(CommandFactory { base }));
        factories.insert("identity".into(), Arc::new(IdentityFactory));
        factories.insert("delegate".into(), Arc::new(DelegateFactory));
        Ok(factories)
    }
    pub fn with_factories(manifest: &Manifest, factories: Factories) -> Result<Self> {
        manifest.validate()?;
        let mut built: BTreeMap<String, (Arc<dyn Driver>, bool)> = BTreeMap::new();
        for provider in &manifest.drivers {
            build_provider(&provider.id, manifest, &factories, &mut built)?;
        }
        let mut handles = BTreeMap::new();
        for provider in &manifest.drivers {
            let (driver, dependencies_granted) = &built[&provider.id];
            for capability in &provider.provides {
                handles.insert(
                    capability.clone(),
                    CapabilityHandle {
                        capability: capability.clone(),
                        provider: provider.id.clone(),
                        resource: provider
                            .resource
                            .clone()
                            .unwrap_or_else(|| capability.clone()),
                        granted: *dependencies_granted && manifest.grants.contains(capability),
                        driver: driver.clone(),
                    },
                );
            }
        }
        Ok(Self {
            handles,
            roots: manifest.requires.iter().cloned().collect(),
        })
    }
}

fn build_provider(
    id: &str,
    manifest: &Manifest,
    factories: &Factories,
    built: &mut BTreeMap<String, (Arc<dyn Driver>, bool)>,
) -> Result<()> {
    if built.contains_key(id) {
        return Ok(());
    }
    let provider = manifest
        .drivers
        .iter()
        .find(|p| p.id == id)
        .context("provider missing after graph validation")?;
    let mut dependencies = BTreeMap::new();
    for capability in &provider.requires {
        let target = manifest
            .provider(capability)
            .context("dependency missing after graph validation")?;
        build_provider(&target.id, manifest, factories, built)?;
        let (driver, dependencies_granted) = &built[&target.id];
        dependencies.insert(
            capability.clone(),
            CapabilityHandle {
                capability: capability.clone(),
                provider: target.id.clone(),
                resource: target
                    .resource
                    .clone()
                    .unwrap_or_else(|| capability.clone()),
                granted: *dependencies_granted && manifest.grants.contains(capability),
                driver: driver.clone(),
            },
        );
    }
    let dependencies_granted = dependencies.values().all(|h| h.granted);
    let factory = factories
        .get(&provider.adapter)
        .context("unknown driver adapter")?;
    let driver = factory.build(&provider.config, Dependencies(dependencies))?;
    built.insert(id.to_owned(), (driver, dependencies_granted));
    Ok(())
}
impl Dispatcher for Registry {
    fn authorized(&self, capability: &str) -> bool {
        self.roots.contains(capability) && self.handles.get(capability).is_some_and(|h| h.granted)
    }
    fn provider(&self, capability: &str) -> Option<&str> {
        self.roots
            .contains(capability)
            .then(|| self.handles.get(capability))
            .flatten()
            .map(|h| h.provider.as_str())
    }
    fn resource(&self, capability: &str) -> Option<&str> {
        self.roots
            .contains(capability)
            .then(|| self.handles.get(capability))
            .flatten()
            .map(|h| h.resource.as_str())
    }
    fn submit(&self, capability: &str, request: Invocation) -> Result<Box<dyn Pending>> {
        ensure!(
            self.authorized(capability),
            "EPERM: capability or dependency is not granted"
        );
        let handle = self.handles[capability].clone();
        ensure!(
            request.lease.resource == handle.resource && request.lease.token > 0,
            "invocation does not carry the selected resource fence"
        );
        let request = serde_json::to_value(request)?;
        let worker = std::thread::Builder::new()
            .name("arcloom-driver".into())
            .spawn(move || handle.invoke(&request))?;
        Ok(Box::new(Worker(worker)))
    }
}
struct Worker(JoinHandle<Result<Value>>);
impl Pending for Worker {
    fn finish(self: Box<Self>) -> Result<Value> {
        self.0
            .join()
            .map_err(|_| anyhow::anyhow!("driver worker panicked"))?
    }
}
struct CommandFactory {
    base: PathBuf,
}
impl AdapterFactory for CommandFactory {
    fn build(&self, config: &Value, _dependencies: Dependencies) -> Result<Arc<dyn Driver>> {
        let mut command: CommandDriver = serde_json::from_value(config.clone())?;
        command.cwd = self.base.join(&command.cwd).canonicalize()?;
        if let Some(executable) = command.argv.first_mut() {
            if !Path::new(executable).is_absolute() {
                *executable = self
                    .base
                    .join(&*executable)
                    .canonicalize()?
                    .to_string_lossy()
                    .into_owned();
            }
        }
        command.validate()?;
        Ok(Arc::new(command))
    }
}
struct IdentityFactory;
struct Identity;
impl AdapterFactory for IdentityFactory {
    fn build(&self, config: &Value, _dependencies: Dependencies) -> Result<Arc<dyn Driver>> {
        ensure!(
            config.as_object().is_some_and(|v| v.is_empty()),
            "identity adapter expects empty configuration"
        );
        Ok(Arc::new(Identity))
    }
}
impl Driver for Identity {
    fn invoke(&self, request: &Value) -> Result<Value> {
        request
            .get("input")
            .cloned()
            .context("identity request requires input")
    }
}
struct DelegateFactory;
struct Delegate(CapabilityHandle);
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct DelegateConfig {
    capability: String,
}
impl AdapterFactory for DelegateFactory {
    fn build(&self, config: &Value, dependencies: Dependencies) -> Result<Arc<dyn Driver>> {
        let config: DelegateConfig = serde_json::from_value(config.clone())?;
        Ok(Arc::new(Delegate(dependencies.get(&config.capability)?)))
    }
}
impl Driver for Delegate {
    fn invoke(&self, request: &Value) -> Result<Value> {
        self.0.invoke(request)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    fn request(input: Value) -> Invocation {
        Invocation {
            run: "test-run".into(),
            action: 1,
            generation: 1,
            input,
            intent: Value::Null,
            lease: crate::ports::LeaseFence {
                resource: "shared-work".into(),
                token: 1,
            },
        }
    }
    fn manifest() -> Manifest {
        serde_json::from_value(json!({"version":2,"component":"unused","intent":{},"requires":["root"],"grants":["root","leaf"],
        "drivers":[{"id":"delegator","resource":"shared-work","provides":["root"],"requires":["leaf"],"adapter":"delegate","config":{"capability":"leaf"}},
        {"id":"leaf-provider","resource":"shared-work","provides":["leaf"],"requires":[],"adapter":"identity","config":{}}],
        "limits":{"max_actions":3,"fuel":1000,"memory_bytes":65536}})).unwrap()
    }
    #[test]
    fn native_dependency_handle_is_usable_without_granting_root_direct_access() {
        let registry = Registry::from_manifest(&manifest()).unwrap();
        assert!(registry.authorized("root"));
        assert!(!registry.authorized("leaf"));
        assert_eq!(registry.provider("root"), Some("delegator"));
        assert_eq!(
            registry
                .submit("root", request(json!({"arbitrary":"payload"})))
                .unwrap()
                .finish()
                .unwrap(),
            json!({"arbitrary":"payload"})
        );
        assert!(registry.submit("leaf", request(Value::Null)).is_err());
    }
    #[test]
    fn declarations_never_amplify_grants() {
        let mut m = manifest();
        m.grants = vec!["root".into()];
        let registry = Registry::from_manifest(&m).unwrap();
        assert!(!registry.authorized("root"));
        assert!(registry.submit("root", request(json!(1))).is_err());
        let mut m = manifest();
        m.drivers[0].requires.clear();
        m.grants.retain(|c| c != "leaf");
        assert!(Registry::from_manifest(&m).is_err());
    }
    struct CustomFactory;
    struct CustomDriver {
        dependency: CapabilityHandle,
    }
    impl AdapterFactory for CustomFactory {
        fn build(&self, _: &Value, dependencies: Dependencies) -> Result<Arc<dyn Driver>> {
            assert!(dependencies.get("undeclared").is_err());
            Ok(Arc::new(CustomDriver {
                dependency: dependencies.get("leaf")?,
            }))
        }
    }
    impl Driver for CustomDriver {
        fn invoke(&self, request: &Value) -> Result<Value> {
            let observed = self.dependency.invoke(request)?;
            Ok(
                json!({"custom_transport":observed,"provider":self.dependency.provider(),"capability":self.dependency.capability()}),
            )
        }
    }
    #[test]
    fn host_registers_a_third_transport_without_kernel_or_graph_changes() {
        let mut m = manifest();
        m.drivers[0].adapter = "custom".into();
        let mut factories = Registry::default_factories(&m).unwrap();
        factories.insert("custom".into(), Arc::new(CustomFactory));
        let registry = Registry::with_factories(&m, factories).unwrap();
        assert_eq!(
            registry
                .submit("root", request(json!([1, 2])))
                .unwrap()
                .finish()
                .unwrap(),
            json!({"custom_transport":[1,2],"provider":"leaf-provider","capability":"leaf"})
        );
    }
    struct FailingFactory;
    struct FailingDriver;
    impl AdapterFactory for FailingFactory {
        fn build(&self, _: &Value, _: Dependencies) -> Result<Arc<dyn Driver>> {
            Ok(Arc::new(FailingDriver))
        }
    }
    impl Driver for FailingDriver {
        fn invoke(&self, _: &Value) -> Result<Value> {
            anyhow::bail!("provider unavailable")
        }
    }
    #[test]
    fn dependency_errors_propagate_through_async_completion() {
        let mut m = manifest();
        m.drivers[1].adapter = "failing".into();
        let mut factories = Registry::default_factories(&m).unwrap();
        factories.insert("failing".into(), Arc::new(FailingFactory));
        let registry = Registry::with_factories(&m, factories).unwrap();
        assert!(registry
            .submit("root", request(Value::Null))
            .unwrap()
            .finish()
            .is_err());
    }
    #[test]
    fn invalid_adapter_configuration_fails_before_workers_start() {
        let mut m = manifest();
        m.drivers[1].adapter = "unknown".into();
        assert!(Registry::from_manifest(&m).is_err());
        let mut m = manifest();
        m.drivers[1].config = json!({"unexpected":true});
        assert!(Registry::from_manifest(&m).is_err());
    }
    struct BlockingFactory(Arc<Blocking>);
    struct Blocking {
        entered: std::sync::mpsc::Sender<()>,
        release: std::sync::Mutex<std::sync::mpsc::Receiver<()>>,
    }
    impl AdapterFactory for BlockingFactory {
        fn build(&self, _: &Value, _: Dependencies) -> Result<Arc<dyn Driver>> {
            Ok(self.0.clone())
        }
    }
    impl Driver for Blocking {
        fn invoke(&self, request: &Value) -> Result<Value> {
            self.entered.send(())?;
            self.release
                .lock()
                .unwrap()
                .recv_timeout(std::time::Duration::from_secs(3))?;
            Ok(request["input"].clone())
        }
    }
    #[test]
    fn submit_returns_pending_before_provider_completion() {
        let (entered_tx, entered_rx) = std::sync::mpsc::channel();
        let (release_tx, release_rx) = std::sync::mpsc::channel();
        let mut m = manifest();
        m.drivers[1].adapter = "blocking".into();
        let mut factories = Registry::default_factories(&m).unwrap();
        factories.insert(
            "blocking".into(),
            Arc::new(BlockingFactory(Arc::new(Blocking {
                entered: entered_tx,
                release: std::sync::Mutex::new(release_rx),
            }))),
        );
        let registry = Registry::with_factories(&m, factories).unwrap();
        let pending = registry
            .submit("root", request(json!({"test":"value"})))
            .unwrap();
        entered_rx
            .recv_timeout(std::time::Duration::from_secs(1))
            .unwrap();
        release_tx.send(()).unwrap();
        assert_eq!(pending.finish().unwrap(), json!({"test":"value"}));
    }
    #[test]
    fn invocation_cannot_change_the_selected_fence_domain() {
        let registry = Registry::from_manifest(&manifest()).unwrap();
        let mut invocation = request(Value::Null);
        invocation.lease.resource = "unrelated".into();
        assert!(registry.submit("root", invocation).is_err());
        let mut invocation = request(Value::Null);
        invocation.lease.token = 0;
        assert!(registry.submit("root", invocation).is_err());
        assert_eq!(registry.resource("root"), Some("shared-work"));
    }
}
