use anyhow::{bail, ensure, Context, Result};
use arcloom_os::{application::WasmApplication, journal::Journal, kernel, model::Manifest};
use fs2::FileExt;
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::{
    fs::{self, File, OpenOptions},
    path::Path,
};

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
    match args.iter().map(String::as_str).collect::<Vec<_>>().as_slice() {
        ["pack",input,output]=> {
            let bytes=wit_component::ComponentEncoder::default().module(&fs::read(input)?)?.validate(true).encode()?;
            fs::write(output,bytes)?;
        }
        ["run",manifest,dir,id]=> {
            let m=Manifest::load(Path::new(manifest))?;
            let bytes=fs::read(&m.component)?;
            let mut digest=Sha256::new();digest.update(serde_json::to_vec(&m)?);digest.update(&bytes);
            let binding=format!("{:x}",digest.finalize());
            let mut app=WasmApplication::new(&bytes,&m.limits)?;
            let _lock=lock(Path::new(dir))?;
            let mut journal=Journal::open(Path::new(dir))?;
            let result=kernel::run(&m,&binding,id,&mut journal,&mut app)?;
            print(&result)?;
            if !matches!(result.status.as_str(),"completed"|"waiting") {return Ok(2);}
        }
        ["inspect",dir,id]=> {
            ensure!(Path::new(dir).join("journal.sqlite").exists(),"unknown state directory");
            let _lock=lock(Path::new(dir))?;
            let j=Journal::open(Path::new(dir))?;
            print(&json!({"run":j.run(id)?,"signals":j.signals(id)?}))?;
        }
        ["context",dir]=> {
            let _lock=lock(Path::new(dir))?;
            let j=Journal::open(Path::new(dir))?;
            print(&j.context(kernel::now()?)?)?;
        }
        ["knowledge-put",dir,input]=> {
            let _lock=lock(Path::new(dir))?;
            let j=Journal::open(Path::new(dir))?;
            j.add_knowledge(&serde_json::from_slice::<Value>(&fs::read(input)?)?)?;
        }
        ["artifact-put",dir,input]=> {
            let _lock=lock(Path::new(dir))?;
            let j=Journal::open(Path::new(dir))?;
            print(&json!({"sha256":j.put_artifact(&fs::read(input)?)?}))?;
        }
        ["signal",dir,id,kind,input]=> {
            let _lock=lock(Path::new(dir))?;
            let j=Journal::open(Path::new(dir))?;
            let id=j.append_signal(id,kind,&serde_json::from_slice::<Value>(&fs::read(input)?)?)?;
            print(&json!({"signal":id}))?;
        }
        _=>bail!("usage: arcloom-os pack INPUT OUTPUT | run MANIFEST STATE_DIR RUN_ID | inspect STATE_DIR RUN_ID | context STATE_DIR | knowledge-put STATE_DIR JSON | artifact-put STATE_DIR FILE | signal STATE_DIR RUN_ID KIND JSON"),
    }
    Ok(0)
}
