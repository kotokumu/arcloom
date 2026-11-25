//! Linux-only bounded JSON transport for explicitly trusted local driver programs.
use anyhow::{bail, ensure, Context, Result};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    collections::BTreeMap,
    io::{Read, Write},
    os::{fd::AsRawFd, unix::process::CommandExt},
    path::{Path, PathBuf},
    process::{Child, Command, ExitStatus, Stdio},
    time::{Duration, Instant},
};

use crate::registry::Driver;

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CommandDriver {
    pub argv: Vec<String>,
    pub cwd: PathBuf,
    pub env: BTreeMap<String, String>,
    pub timeout_ms: u64,
    pub max_output_bytes: usize,
}

impl CommandDriver {
    pub fn validate(&self) -> Result<()> {
        let executable = self.argv.first().context("driver argv must not be empty")?;
        ensure!(
            Path::new(executable).is_absolute(),
            "driver executable must be absolute"
        );
        ensure!(
            self.cwd.is_absolute() && self.cwd.is_dir(),
            "driver cwd must be an existing absolute directory"
        );
        ensure!(
            (1..=86_400_000).contains(&self.timeout_ms),
            "driver timeout must be between 1ms and 24h"
        );
        ensure!(
            (1..=64 * 1024 * 1024).contains(&self.max_output_bytes),
            "driver output limit must be between 1 byte and 64MiB"
        );
        ensure!(
            self.argv.iter().all(|s| !s.contains('\0')),
            "driver argv contains NUL"
        );
        ensure!(
            self.env
                .iter()
                .all(|(k, v)| !k.is_empty() && !k.contains(['=', '\0']) && !v.contains('\0')),
            "driver environment contains an invalid entry"
        );
        Ok(())
    }
}

/// Kills the owned process group on every exit path, then reaps its direct child.
/// Grandchildren are killed, but reaping them remains their parent's/init's duty.
struct OwnedProcess {
    child: Child,
    reaped: bool,
}
impl OwnedProcess {
    fn exited(&mut self) -> Result<bool> {
        if self.reaped {
            return Ok(true);
        }
        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        // WNOWAIT keeps the exited leader waitable, retaining ownership of its
        // numeric PID until finish kills its process group and reaps it.
        let result = unsafe {
            libc::waitid(
                libc::P_PID,
                self.child.id() as libc::id_t,
                &mut info,
                libc::WEXITED | libc::WNOHANG | libc::WNOWAIT,
            )
        };
        if result == -1 {
            let error = std::io::Error::last_os_error();
            if error.kind() == std::io::ErrorKind::Interrupted {
                return Ok(false);
            }
            // An outside reaper has broken ownership. Do not signal a PID that
            // might now belong to someone else, even while reporting failure.
            if error.raw_os_error() == Some(libc::ECHILD) {
                self.reaped = true;
            }
            return Err(error).context("driver process observation failed");
        }
        Ok(unsafe { info.si_pid() } != 0)
    }

    fn finish(&mut self) -> Result<ExitStatus> {
        if !self.reaped {
            // The unreaped child retains this PID: the group cannot be confused
            // with unrelated work created after PID reuse.
            unsafe {
                libc::kill(-(self.child.id() as libc::pid_t), libc::SIGKILL);
            }
            let _ = self.child.kill();
            let result = self.child.wait();
            // The cleanup signal is one-shot. Never signal this numeric group
            // again after wait, including if an external reaper caused ECHILD.
            self.reaped = true;
            return result.context("driver process wait failed");
        }
        self.child.wait().context("driver process wait failed")
    }
}
impl Drop for OwnedProcess {
    fn drop(&mut self) {
        if !self.reaped {
            let _ = self.finish();
        }
    }
}

fn nonblocking(fd: &impl AsRawFd) -> Result<()> {
    let fd = fd.as_raw_fd();
    // fcntl preserves existing descriptor flags and never changes ownership.
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    ensure!(flags >= 0, "cannot inspect driver pipe flags");
    ensure!(
        unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } >= 0,
        "cannot make driver pipe nonblocking"
    );
    Ok(())
}

impl Driver for CommandDriver {
    fn invoke(&self, request: &Value) -> Result<Value> {
        self.validate()?;
        let mut input = serde_json::to_vec(request)?;
        input.push(b'\n');
        ensure!(
            input.len() <= self.max_output_bytes,
            "driver request exceeds configured byte limit"
        );
        let started = Instant::now();
        let parent_pid = unsafe { libc::getpid() };
        let mut command = Command::new(&self.argv[0]);
        command
            .args(&self.argv[1..])
            .current_dir(&self.cwd)
            .env_clear()
            .envs(&self.env)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .process_group(0);
        // Linux kills the direct child if this host dies. This is not a claim of
        // orphan reclamation for grandchildren; a process sandbox is deferred.
        unsafe {
            command.pre_exec(move || {
                if libc::prctl(libc::PR_SET_PDEATHSIG, libc::SIGKILL) != 0 {
                    return Err(std::io::Error::last_os_error());
                }
                if libc::getppid() != parent_pid {
                    libc::_exit(127);
                }
                Ok(())
            });
        }
        let child = command.spawn().context("driver process could not start")?;
        let mut owned = OwnedProcess {
            child,
            reaped: false,
        };
        let mut writer = owned.child.stdin.take();
        let mut reader = owned.child.stdout.take().context("driver stdout missing")?;
        nonblocking(writer.as_ref().context("driver stdin missing")?)?;
        nonblocking(&reader)?;
        let mut written = 0;
        let mut output = Vec::new();
        let mut eof = false;
        let mut status = None;
        loop {
            ensure!(
                started.elapsed() < Duration::from_millis(self.timeout_ms),
                "driver timeout"
            );
            if let Some(pipe) = writer.as_mut() {
                match pipe.write(&input[written..]) {
                    Ok(n) => written += n,
                    Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                    Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                    Err(e) if e.kind() == std::io::ErrorKind::BrokenPipe => written = input.len(),
                    Err(_) => bail!("driver stdin write failed"),
                }
                if written == input.len() {
                    writer.take();
                }
            }
            let mut chunk = [0u8; 8192];
            loop {
                ensure!(
                    started.elapsed() < Duration::from_millis(self.timeout_ms),
                    "driver timeout"
                );
                match reader.read(&mut chunk) {
                    Ok(0) => {
                        eof = true;
                        break;
                    }
                    Ok(n) => {
                        ensure!(
                            output.len() + n <= self.max_output_bytes,
                            "driver response exceeds configured byte limit"
                        );
                        output.extend_from_slice(&chunk[..n]);
                    }
                    Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => break,
                    Err(e) if e.kind() == std::io::ErrorKind::Interrupted => continue,
                    Err(_) => bail!("driver stdout read failed"),
                }
            }
            if status.is_none() && owned.exited()? {
                status = Some(owned.finish()?);
            }
            if let Some(exit) = status {
                ensure!(exit.success(), "driver exited unsuccessfully");
                if eof {
                    break;
                }
            }
            std::thread::sleep(Duration::from_millis(2));
        }
        serde_json::from_slice(&output).context("driver response is not one JSON value")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    use std::time::{Duration, Instant};

    fn python(code: &str) -> CommandDriver {
        CommandDriver {
            argv: vec!["/usr/bin/python3".into(), "-c".into(), code.into()],
            cwd: PathBuf::from("/tmp"),
            env: BTreeMap::new(),
            timeout_ms: 2_000,
            max_output_bytes: 4096,
        }
    }

    #[test]
    fn interchangeable_programs_accept_opaque_json() {
        let first = python(
            "import json,sys; r=json.load(sys.stdin); print(json.dumps({'result':r['input']}))",
        );
        let second = python("import json,sys; r=json.load(sys.stdin); print(json.dumps({'result':r['input'][::-1]}))");
        assert_eq!(
            first.invoke(&json!({"input":"ab"})).unwrap(),
            json!({"result":"ab"})
        );
        assert_eq!(
            second.invoke(&json!({"input":"ab"})).unwrap(),
            json!({"result":"ba"})
        );
    }

    #[test]
    fn explicit_environment_and_literal_argv() {
        let mut command = python("import os,json,sys; print(json.dumps({'secret':os.environ.get('HOME'),'value':os.environ.get('DRIVER_VALUE'),'arg':sys.argv[1]}))");
        command
            .env
            .insert("DRIVER_VALUE".into(), "configured".into());
        command
            .argv
            .push("$(touch /tmp/arcloom-must-not-execute)".into());
        assert_eq!(
            command.invoke(&json!({})).unwrap(),
            json!({"secret":null,"value":"configured","arg":"$(touch /tmp/arcloom-must-not-execute)"})
        );
    }

    #[test]
    fn rejects_invalid_bindings_before_spawn() {
        let mut command = python("raise Exception('must not execute')");
        command.argv.clear();
        assert!(command.validate().is_err());
        command.argv.push("python3".into());
        assert!(command.validate().is_err());
        command.argv[0] = "/usr/bin/python3".into();
        command.cwd = PathBuf::from("relative");
        assert!(command.validate().is_err());
        command.cwd = PathBuf::from("/tmp");
        command.timeout_ms = 0;
        assert!(command.validate().is_err());
        command.timeout_ms = u64::MAX;
        assert!(command.validate().is_err());
        command.timeout_ms = 100;
        command.max_output_bytes = 0;
        assert!(command.validate().is_err());
        command.max_output_bytes = usize::MAX;
        assert!(command.validate().is_err());
    }

    #[test]
    fn rejects_failure_and_malformed_response_without_stderr_disclosure() {
        for code in [
            "import sys; sys.stderr.write('private credential'); sys.exit(3)",
            "print('invalid')",
        ] {
            let error = python(code).invoke(&json!({})).unwrap_err().to_string();
            assert!(!error.contains("private credential"));
        }
    }

    #[test]
    fn bounds_response_and_request() {
        let mut command = python("print('x'*1000000)");
        command.max_output_bytes = 128;
        assert!(command
            .invoke(&json!({}))
            .unwrap_err()
            .to_string()
            .contains("limit"));
        assert!(command
            .invoke(&json!({"input":"x".repeat(129)}))
            .unwrap_err()
            .to_string()
            .contains("limit"));
    }

    #[test]
    fn blocked_input_obeys_deadline_and_reaps_child() {
        let mut command = python("import time; time.sleep(30)");
        command.timeout_ms = 100;
        command.max_output_bytes = 2 * 1024 * 1024;
        let start = Instant::now();
        assert!(command
            .invoke(&json!({"input":"x".repeat(1024*1024)}))
            .unwrap_err()
            .to_string()
            .contains("timeout"));
        assert!(start.elapsed() < Duration::from_secs(3));
    }

    #[test]
    fn timeout_stops_descendants_in_process_group() {
        let dir = tempfile::tempdir().unwrap();
        let pid_file = dir.path().join("pid");
        let mut command = python("import subprocess,sys,time; p=subprocess.Popen(['/usr/bin/python3','-c','import time; time.sleep(30)']); open(sys.argv[1],'w').write(str(p.pid)); time.sleep(30)");
        command.argv.push(pid_file.display().to_string());
        command.timeout_ms = 200;
        assert!(command.invoke(&json!({})).is_err());
        let pid = std::fs::read_to_string(pid_file).unwrap();
        let deadline = Instant::now() + Duration::from_secs(1);
        loop {
            match std::fs::read_to_string(format!("/proc/{pid}/stat")) {
                Err(e) if e.kind() == std::io::ErrorKind::NotFound => break,
                Ok(stat) if stat.split_whitespace().nth(2) == Some("Z") => break,
                _ => {
                    assert!(
                        Instant::now() < deadline,
                        "driver descendant survived timeout"
                    );
                    std::thread::sleep(Duration::from_millis(2));
                }
            }
        }
    }
    #[test]
    fn observing_exit_preserves_pid_ownership_until_group_cleanup() {
        let child = Command::new("/usr/bin/python3")
            .args(["-c", "pass"])
            .process_group(0)
            .spawn()
            .unwrap();
        let pid = child.id() as libc::pid_t;
        let mut owned = OwnedProcess {
            child,
            reaped: false,
        };
        let deadline = Instant::now() + Duration::from_secs(2);
        while !owned.exited().unwrap() {
            assert!(Instant::now() < deadline);
            std::thread::sleep(Duration::from_millis(2));
        }
        // The leader must still be our waitable child: its PID cannot be reused
        // before the process-group signal has been sent.
        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        assert_eq!(
            unsafe {
                libc::waitid(
                    libc::P_PID,
                    pid as libc::id_t,
                    &mut info,
                    libc::WEXITED | libc::WNOWAIT | libc::WNOHANG,
                )
            },
            0
        );
        assert_eq!(unsafe { info.si_pid() }, pid);
        assert!(owned.finish().unwrap().success());
        assert_eq!(
            unsafe {
                libc::waitid(
                    libc::P_PID,
                    pid as libc::id_t,
                    &mut info,
                    libc::WEXITED | libc::WNOWAIT | libc::WNOHANG,
                )
            },
            -1
        );
        assert_eq!(
            std::io::Error::last_os_error().raw_os_error(),
            Some(libc::ECHILD)
        );
        // Repeated finish/drop must use the cached status, without group signals.
        assert!(owned.finish().unwrap().success());
    }

    #[test]
    fn successful_driver_exit_also_stops_lingering_descendants() {
        let command = python("import subprocess,os,json; p=subprocess.Popen(['/usr/bin/python3','-c','import time; time.sleep(30)']); print(json.dumps({'pid':os.getpid(),'descendant':p.pid}),flush=True)");
        let result = command.invoke(&json!({})).unwrap();
        let pid = result["pid"].as_i64().unwrap() as libc::pid_t;
        let mut status = 0;
        assert_eq!(
            unsafe { libc::waitpid(pid, &mut status, libc::WNOHANG) },
            -1
        );
        assert_eq!(
            std::io::Error::last_os_error().raw_os_error(),
            Some(libc::ECHILD)
        );
        let descendant = result["descendant"].as_i64().unwrap();
        let deadline = Instant::now() + Duration::from_secs(1);
        loop {
            match std::fs::read_to_string(format!("/proc/{descendant}/stat")) {
                Err(e) if e.kind() == std::io::ErrorKind::NotFound => break,
                Ok(stat) if stat.split_whitespace().nth(2) == Some("Z") => break,
                _ => {
                    assert!(
                        Instant::now() < deadline,
                        "descendant survived successful driver exit"
                    );
                    std::thread::sleep(Duration::from_millis(2));
                }
            }
        }
    }
}
