use crate::drivers::CommandDriver;
use anyhow::{bail, ensure, Context, Result};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    collections::BTreeSet,
    path::{Path, PathBuf},
};

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Limits {
    pub max_actions: u64,
    pub fuel: u64,
    pub memory_bytes: usize,
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Provider {
    pub id: String,
    pub kind: String,
    pub provides: Vec<String>,
    pub command: CommandDriver,
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Manifest {
    pub version: u32,
    pub component: PathBuf,
    pub intent: Value,
    pub requires: Vec<String>,
    pub grants: Vec<String>,
    pub drivers: Vec<Provider>,
    pub limits: Limits,
}
#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Transition {
    pub state: Value,
    pub effect: Effect,
}
#[derive(Debug, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "snake_case", deny_unknown_fields)]
pub enum Effect {
    Wait,
    Complete,
    Execute { capability: String, input: Value },
    Evaluate { capability: String, input: Value },
}
impl Manifest {
    pub fn load(path: &Path) -> Result<Self> {
        let path = path.canonicalize()?;
        let base = path.parent().context("manifest directory")?;
        let mut m: Self = serde_json::from_slice(&std::fs::read(&path)?)?;
        m.component = base.join(&m.component).canonicalize()?;
        for d in &mut m.drivers {
            d.command.cwd = base.join(&d.command.cwd).canonicalize()?;
            if let Some(exe) = d.command.argv.first_mut() {
                if !Path::new(exe).is_absolute() {
                    *exe = base
                        .join(&*exe)
                        .canonicalize()?
                        .to_string_lossy()
                        .into_owned();
                }
            }
        }
        m.validate()?;
        Ok(m)
    }
    pub fn validate(&self) -> Result<()> {
        ensure!(self.version == 1, "unsupported manifest version");
        ensure!(
            self.limits.max_actions > 0 && self.limits.fuel > 0 && self.limits.memory_bytes > 0,
            "limits must be positive"
        );
        let mut ids = BTreeSet::new();
        let mut capabilities = BTreeSet::new();
        for p in &self.drivers {
            ensure!(
                !p.id.is_empty() && ids.insert(&p.id),
                "duplicate or empty provider id"
            );
            ensure!(
                matches!(p.kind.as_str(), "execution" | "feedback"),
                "unknown driver kind"
            );
            ensure!(!p.provides.is_empty(), "provider has no capabilities");
            p.command.validate()?;
            for c in &p.provides {
                ensure!(
                    !c.is_empty() && capabilities.insert(c),
                    "ambiguous or empty capability: {c}"
                );
            }
        }
        let mut required = BTreeSet::new();
        for c in &self.requires {
            ensure!(
                required.insert(c) && capabilities.contains(c),
                "duplicate or unresolved requirement: {c}"
            );
        }
        let mut granted = BTreeSet::new();
        for c in &self.grants {
            ensure!(
                granted.insert(c) && required.contains(c),
                "grant is duplicate or not required: {c}"
            );
        }
        Ok(())
    }
    pub fn authorize(&self, capability: &str, kind: &str) -> Result<&Provider> {
        ensure!(
            self.requires.iter().any(|c| c == capability)
                && self.grants.iter().any(|c| c == capability),
            "EPERM: capability not granted"
        );
        for p in &self.drivers {
            if p.kind == kind && p.provides.iter().any(|c| c == capability) {
                return Ok(p);
            }
        }
        bail!("EPERM: no provider of required kind")
    }
}
