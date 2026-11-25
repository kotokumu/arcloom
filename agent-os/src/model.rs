//! Declarative capability graph. Concrete transport configuration is opaque here.
use anyhow::{ensure, Context, Result};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    collections::{BTreeMap, BTreeSet},
    path::{Path, PathBuf},
};

pub const HOST_READ_CAPABILITIES: &[&str] = &["knowledge.read", "artifact.read"];
fn default_lease_ttl() -> i64 {
    60
}
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Limits {
    pub max_actions: u64,
    pub fuel: u64,
    pub memory_bytes: usize,
    #[serde(default = "default_lease_ttl")]
    pub lease_ttl_seconds: i64,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Provider {
    pub id: String,
    pub provides: Vec<String>,
    #[serde(default)]
    pub requires: Vec<String>,
    #[serde(default)]
    pub resource: Option<String>,
    pub adapter: String,
    pub config: Value,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Manifest {
    pub version: u32,
    pub component: PathBuf,
    pub intent: Value,
    pub requires: Vec<String>,
    pub grants: Vec<String>,
    pub drivers: Vec<Provider>,
    pub limits: Limits,
    #[serde(default)]
    pub coalescing: Vec<String>,
    /// Established by the loader, included in the immutable run binding.
    #[serde(skip_deserializing, default)]
    pub base_dir: PathBuf,
}
impl Manifest {
    pub fn load(path: &Path) -> Result<Self> {
        let path = path.canonicalize()?;
        let base = path.parent().context("manifest directory")?;
        let value: Value = serde_json::from_slice(&std::fs::read(&path)?)?;
        ensure!(
            value.get("base_dir").is_none(),
            "base_dir is established by the manifest loader"
        );
        let mut m: Self = serde_json::from_value(value)?;
        m.component = base.join(&m.component).canonicalize()?;
        m.base_dir = base.to_path_buf();
        m.validate()?;
        Ok(m)
    }
    pub fn provider(&self, capability: &str) -> Option<&Provider> {
        self.drivers
            .iter()
            .find(|p| p.provides.iter().any(|c| c == capability))
    }
    pub fn validate(&self) -> Result<()> {
        ensure!(self.version==2,"manifest version 2 required: migrate providers to adapter/config, requires and resource declarations");
        ensure!(
            self.limits.max_actions > 0 && self.limits.fuel > 0 && self.limits.memory_bytes > 0,
            "limits must be positive"
        );
        ensure!(
            (1..=86400).contains(&self.limits.lease_ttl_seconds),
            "lease TTL must be between 1 second and 24 hours"
        );
        let mut ids = BTreeSet::new();
        let mut capabilities = BTreeMap::new();
        for (index, p) in self.drivers.iter().enumerate() {
            ensure!(
                !p.id.trim().is_empty() && ids.insert(&p.id),
                "duplicate or empty provider id"
            );
            ensure!(!p.adapter.trim().is_empty(), "empty adapter name");
            ensure!(
                p.resource.as_ref().is_none_or(|s| !s.trim().is_empty()),
                "empty provider resource"
            );
            ensure!(!p.provides.is_empty(), "provider has no capabilities");
            for c in &p.provides {
                ensure!(
                    !c.trim().is_empty()
                        && !HOST_READ_CAPABILITIES.contains(&c.as_str())
                        && capabilities.insert(c, index).is_none(),
                    "ambiguous, reserved or empty capability: {c}"
                );
            }
        }
        for p in &self.drivers {
            let mut seen = BTreeSet::new();
            for c in &p.requires {
                ensure!(
                    seen.insert(c) && capabilities.contains_key(c),
                    "duplicate or unresolved provider requirement: {c}"
                );
            }
        }
        // One kernel reservation covers a composed provider invocation. Every
        // dependency must remain in that same resource/fencing domain.
        for p in &self.drivers {
            for capability in &p.provides {
                let resource = p.resource.as_deref().unwrap_or(capability);
                for dependency in &p.requires {
                    let target = self
                        .provider(dependency)
                        .context("dependency provider missing")?;
                    ensure!(resource==target.resource.as_deref().unwrap_or(dependency),"dependency edge crosses resource domains; declare a shared resource on composed providers");
                }
            }
        }
        // Validate every registered provider, including currently unused ones.
        let mut visited = BTreeSet::new();
        let mut visiting = BTreeSet::new();
        for index in 0..self.drivers.len() {
            self.visit(index, &capabilities, &mut visiting, &mut visited)?;
        }
        let mut root = BTreeSet::new();
        let mut reachable = BTreeSet::new();
        for c in &self.requires {
            ensure!(
                root.insert(c)
                    && (HOST_READ_CAPABILITIES.contains(&c.as_str())
                        || capabilities.contains_key(c)),
                "duplicate or unresolved application requirement: {c}"
            );
            self.collect(c, &mut reachable);
        }
        let mut grants = BTreeSet::new();
        for c in &self.grants {
            ensure!(
                grants.insert(c) && reachable.contains(c),
                "grant is duplicate or outside required dependency closure: {c}"
            );
        }
        let mut modes = BTreeSet::new();
        for kind in &self.coalescing {
            ensure!(
                !kind.trim().is_empty() && modes.insert(kind),
                "duplicate or empty coalescing kind"
            );
        }
        Ok(())
    }
    fn visit<'a>(
        &'a self,
        index: usize,
        caps: &BTreeMap<&'a String, usize>,
        visiting: &mut BTreeSet<usize>,
        visited: &mut BTreeSet<usize>,
    ) -> Result<()> {
        if visited.contains(&index) {
            return Ok(());
        }
        ensure!(
            visiting.insert(index),
            "provider dependency cycle at {}",
            self.drivers[index].id
        );
        for c in &self.drivers[index].requires {
            self.visit(caps[c], caps, visiting, visited)?;
        }
        visiting.remove(&index);
        visited.insert(index);
        Ok(())
    }
    fn collect(&self, cap: &str, result: &mut BTreeSet<String>) {
        if !result.insert(cap.to_owned()) {
            return;
        }
        if let Some(p) = self.provider(cap) {
            for dependency in &p.requires {
                self.collect(dependency, result);
            }
        }
    }
}
