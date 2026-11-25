//! The local host can advance a second task while the first driver's ticket is
//! outstanding. These tests drive the same kernel, mailbox and lease paths as the CLI.
use anyhow::{ensure, Result};
use arcloom_os::{
    journal::Journal,
    kernel::{Kernel, RunPolicy, Step},
    ports::{Application, Dispatcher, Effect, Invocation, Pending, Syscalls, Transition},
    storage::{KnowledgeStore, RunStore, Signal},
};
use serde_json::{json, Value};
use std::{cell::Cell, rc::Rc};

const CAP: &str = "layout.preview";

struct Preview {
    submitted: Cell<usize>,
    grant: bool,
}
struct PreviewResult(Value);
impl Pending for PreviewResult {
    fn finish(self: Box<Self>) -> Result<Value> {
        Ok(self.0)
    }
}
impl Dispatcher for Preview {
    fn authorized(&self, cap: &str) -> bool {
        self.grant && cap == CAP
    }
    fn resource(&self, cap: &str) -> Option<&str> {
        (cap == CAP).then_some("workspace")
    }
    fn provider(&self, cap: &str) -> Option<&str> {
        (cap == CAP).then_some("local-preview")
    }
    fn submit(&self, cap: &str, request: Invocation) -> Result<Box<dyn Pending>> {
        ensure!(self.authorized(cap), "ungranted");
        self.submitted.set(self.submitted.get() + 1);
        Ok(Box::new(PreviewResult(
            json!({"token":request.lease.token}),
        )))
    }
}
struct NoContext;
impl KnowledgeStore for NoContext {
    fn context(&self, _: i64) -> Result<Value> {
        panic!("kernel attempted unsolicited knowledge read")
    }
}
struct Render;
impl Application for Render {
    fn on_signal(&mut self, _: &Value, signal: &Signal, _: Rc<dyn Syscalls>) -> Result<Transition> {
        let effect = if signal.kind == CAP {
            Effect::Complete
        } else {
            Effect::Invoke {
                capability: CAP.into(),
                input: signal.payload.clone(),
            }
        };
        Ok(Transition {
            state: signal.payload.clone(),
            effect,
        })
    }
}
fn kernel<'a>(db: &'a Rc<Journal>, dispatcher: &'a Preview) -> Kernel<'a> {
    Kernel {
        runs: db.as_ref(),
        inbox: db.as_ref(),
        leases: db.as_ref(),
        knowledge: Rc::new(NoContext),
        artifacts: db.clone(),
        dispatch: dispatcher,
    }
}
fn policy(grants: &[String]) -> RunPolicy<'_> {
    RunPolicy {
        grants,
        max_actions: 1,
        lease_ttl_seconds: 10,
    }
}
fn task(db: &Journal, id: &str) -> Result<()> {
    db.open_run(id, "binding", &json!(id))?;
    db.prepare_mailbox(id, &[])?;
    Ok(())
}
#[test]
fn alternate_dispatcher_finishes_later_and_another_run_cannot_steal_its_lease() -> Result<()> {
    let dir = tempfile::tempdir()?;
    let db = Rc::new(Journal::open(dir.path())?);
    task(&db, "first")?;
    task(&db, "second")?;
    let driver = Preview {
        submitted: Cell::new(0),
        grant: true,
    };
    let k = kernel(&db, &driver);
    let Step::Submitted(ticket) = k.step("first", &policy(&[CAP.into()]), 0, &mut Render)? else {
        panic!("missing ticket")
    };
    // First ticket hasn't been collected. No command runner, blocking wait, or context read.
    assert!(matches!(
        k.step("second", &policy(&[CAP.into()]), 100, &mut Render)?,
        Step::Busy { .. }
    ));
    assert_eq!(db.run("second")?.cursor, 0);
    assert_eq!(driver.submitted.get(), 1);
    let (run, action, cap, work) = ticket.into_parts();
    let output = work.finish()?;
    db.complete_action(&run, action, &cap, &output)?;
    let Step::Idle(result) = k.step("first", &policy(&[CAP.into()]), 101, &mut Render)? else {
        panic!("not complete")
    };
    assert_eq!(result.status, "completed");
    assert!(matches!(
        k.step("second", &policy(&[CAP.into()]), 101, &mut Render)?,
        Step::Submitted(_)
    ));
    assert!(db.complete_action("first", action, &cap, &output).is_err());
    Ok(())
}
#[test]
fn restart_fences_uncertain_work_and_preserves_unacknowledged_competitor() -> Result<()> {
    let dir = tempfile::tempdir()?;
    let driver = Preview {
        submitted: Cell::new(0),
        grant: true,
    };
    {
        let db = Rc::new(Journal::open(dir.path())?);
        task(&db, "lost")?;
        assert!(matches!(
            kernel(&db, &driver).step("lost", &policy(&[CAP.into()]), 10, &mut Render)?,
            Step::Submitted(_)
        ));
    }
    let db = Rc::new(Journal::open(dir.path())?);
    assert!(db.open_run("lost", "changed", &json!("lost")).is_err());
    assert!(db.recover("lost")?);
    assert_eq!(db.run("lost")?.status, "uncertain");
    task(&db, "competing")?;
    assert!(matches!(
        kernel(&db, &driver).step("competing", &policy(&[CAP.into()]), 100, &mut Render)?,
        Step::Busy { .. }
    ));
    assert_eq!(driver.submitted.get(), 1);
    Ok(())
}
#[test]
fn denial_before_submission_leaves_no_reserved_action() -> Result<()> {
    let dir = tempfile::tempdir()?;
    let db = Rc::new(Journal::open(dir.path())?);
    task(&db, "denied")?;
    let driver = Preview {
        submitted: Cell::new(0),
        grant: false,
    };
    let Step::Idle(result) =
        kernel(&db, &driver).step("denied", &policy(&[CAP.into()]), 0, &mut Render)?
    else {
        panic!("unexpected ticket")
    };
    assert_eq!(result.status, "denied");
    assert_eq!(result.actions, 0);
    assert_eq!(driver.submitted.get(), 0);
    Ok(())
}

struct Observe;
impl Application for Observe {
    fn on_signal(&mut self, _: &Value, s: &Signal, _: Rc<dyn Syscalls>) -> Result<Transition> {
        Ok(Transition {
            state: s.payload.clone(),
            effect: Effect::Wait,
        })
    }
}
#[test]
fn kernel_commits_coalesced_delivery_without_skipping_an_interleaved_queue_event() -> Result<()> {
    let dir = tempfile::tempdir()?;
    let db = Rc::new(Journal::open(dir.path())?);
    db.open_run("events", "binding", &json!({}))?;
    db.prepare_mailbox("events", &["timer".into()])?;
    db.append_signal("events", "timer", &json!({"target":"same","n":1}))?;
    db.append_signal("events", "human_steer", &json!({"note":"keep"}))?;
    db.append_signal("events", "timer", &json!({"target":"same","n":2}))?;
    let driver = Preview {
        submitted: Cell::new(0),
        grant: false,
    };
    let k = kernel(&db, &driver);
    let mut seen = Vec::new();
    for _ in 0..3 {
        let Step::Idle(run) = k.step("events", &policy(&[CAP.into()]), 0, &mut Observe)? else {
            panic!("unexpected dispatch")
        };
        seen.push(run.state);
    }
    assert_eq!(
        seen,
        vec![
            json!({}),
            json!({"target":"same","n":2}),
            json!({"note":"keep"})
        ]
    );
    Ok(())
}
