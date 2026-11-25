use anyhow::{bail, ensure, Context, Result};
use arcloom_os::{
    application::WasmApplication,
    journal::Journal,
    kernel::{Kernel, RunPolicy, Step},
    model::Manifest,
    registry::Registry,
    storage::RunStore,
};
use fs2::FileExt;
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::{
    fs::{self, File, OpenOptions},
    path::Path,
    time::{SystemTime, UNIX_EPOCH},
};

fn now() -> Result<i64> {
    Ok(SystemTime::now()
        .duration_since(UNIX_EPOCH)?
        .as_secs()
        .try_into()?)
}

fn lock(dir: &Path) -> Result<File> {
    fs::create_dir_all(dir)?;
    let file = OpenOptions::new()
        .create(true)
        .truncate(false)
        .read(true)
        .write(true)
        .open(dir.join("host.lock"))?;
    file.try_lock_exclusive()
        .context("state directory is already in use")?;
    Ok(file)
}

fn print(value: &impl serde::Serialize) -> Result<()> {
    println!("{}", serde_json::to_string_pretty(value)?);
    Ok(())
}

fn run(manifest: &Path, dir: &Path, id: &str) -> Result<u8> {
    ensure!(!id.trim().is_empty(), "empty run id");
    let m = Manifest::load(manifest)?;
    let dispatcher = Registry::from_manifest(&m)?;
    let bytes = fs::read(&m.component)?;
    let mut digest = Sha256::new();
    digest.update(serde_json::to_vec(&m)?);
    digest.update(&bytes);
    let binding = format!("{:x}", digest.finalize());
    let mut app = WasmApplication::new(&bytes, &m.limits)?;
    let _lock = lock(dir)?;
    let journal = std::rc::Rc::new(Journal::open(dir)?);
    journal.open_run(id, &binding, &m.intent)?;
    journal.prepare_mailbox(id, &m.coalescing)?;
    journal.recover(id)?;
    let kernel = Kernel {
        runs: journal.as_ref(),
        inbox: journal.as_ref(),
        leases: journal.as_ref(),
        knowledge: journal.clone(),
        artifacts: journal.clone(),
        dispatch: &dispatcher,
    };
    let policy = RunPolicy {
        grants: &m.grants,
        max_actions: m.limits.max_actions,
        lease_ttl_seconds: m.limits.lease_ttl_seconds,
    };
    loop {
        match kernel.step(id, &policy, now()?, &mut app)? {
            Step::Idle(result) => {
                print(&result)?;
                return Ok(
                    if matches!(result.status.as_str(), "completed" | "waiting") {
                        0
                    } else {
                        2
                    },
                );
            }
            Step::Busy { resource } => {
                bail!("resource busy: {resource}; the input remains pending")
            }
            Step::Submitted(ticket) => {
                let (run, action, capability, pending) = ticket.into_parts();
                // This local CLI waits. A different host can service other runs before collecting.
                match pending.finish() {
                    Ok(value) => journal.complete_action(&run, action, &capability, &value)?,
                    Err(_) => {
                        journal.recover(&run)?;
                    }
                }
            }
        }
    }
}

fn main() -> std::process::ExitCode {
    match cli() {
        Ok(code) => std::process::ExitCode::from(code),
        Err(error) => {
            eprintln!("arcloom-os: {error:#}");
            std::process::ExitCode::FAILURE
        }
    }
}

fn cli() -> Result<u8> {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let args: Vec<&str> = args.iter().map(String::as_str).collect();
    match args.as_slice() {
        ["pack", input, output] => {
            let bytes = wit_component::ComponentEncoder::default().module(&fs::read(input)?)?.validate(true).encode()?;
            fs::write(output, bytes)?;
        }
        ["run", manifest, dir, id] => return run(Path::new(manifest), Path::new(dir), id),
        ["inspect", dir, id] => {
            ensure!(Path::new(dir).join("journal.sqlite").exists(), "unknown state directory");
            let _lock = lock(Path::new(dir))?;
            let journal = Journal::open(Path::new(dir))?;
            print(&json!({"run":journal.run(id)?,"signals":journal.signals(id)?,"actions":journal.actions(id)?}))?;
        }
        ["context", dir] => {
            let _lock = lock(Path::new(dir))?;
            print(&Journal::open(Path::new(dir))?.context(now()?)?)?;
        }
        ["knowledge-put", dir, input] => {
            let _lock = lock(Path::new(dir))?;
            Journal::open(Path::new(dir))?.add_knowledge(&serde_json::from_slice::<Value>(&fs::read(input)?)?)?;
        }
        ["lease-reconcile", dir, id, resource, token] => {
            let _lock = lock(Path::new(dir))?;
            let journal = Journal::open(Path::new(dir))?;
            journal.reconcile_lease(resource, id, token.parse()?)?;
        }
        ["artifact-put", dir, input] => {
            let _lock = lock(Path::new(dir))?;
            print(&json!({"sha256":Journal::open(Path::new(dir))?.put_artifact(&fs::read(input)?)?}))?;
        }
        ["signal", dir, id, kind, input] => {
            let _lock = lock(Path::new(dir))?;
            let seq = Journal::open(Path::new(dir))?.append_signal(id,kind,&serde_json::from_slice::<Value>(&fs::read(input)?)?)?;
            print(&json!({"signal":seq}))?;
        }
        _ => bail!("usage: arcloom-os pack INPUT OUTPUT | run MANIFEST STATE_DIR RUN_ID | inspect STATE_DIR RUN_ID | context STATE_DIR | knowledge-put STATE_DIR JSON | lease-reconcile STATE_DIR RUN_ID RESOURCE TOKEN | artifact-put STATE_DIR FILE | signal STATE_DIR RUN_ID KIND JSON"),
    }
    Ok(0)
}
