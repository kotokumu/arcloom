use anyhow::Result;
use arcloom_os::{
    application::Application,
    journal::Journal,
    kernel::run,
    model::{Effect, Manifest, Transition},
};
use serde_json::{json, Value};
struct Stop;
impl Application for Stop {
    fn on_signal(&mut self, _state: &Value, signal: &Value) -> Result<Transition> {
        Ok(Transition {
            state: json!({"seen":signal["kind"]}),
            effect: Effect::Complete,
        })
    }
}
fn manifest(grants: Value) -> Manifest {
    serde_json::from_value(json!({"version":1,"component":"unused","intent":{"goal":"test"},"requires":["work"],"grants":grants,
    "drivers":[{"id":"test","kind":"execution","provides":["work"],"command":{"argv":["/usr/bin/python3","-c","print('{}')"],"cwd":"/tmp","env":{},"timeout_ms":1000,"max_output_bytes":4096}}],
    "limits":{"max_actions":1,"fuel":1000000,"memory_bytes":16777216}})).unwrap()
}
#[test]
fn app_can_complete_without_any_driver() {
    let dir = tempfile::tempdir().unwrap();
    let mut j = Journal::open(dir.path()).unwrap();
    let result = run(&manifest(json!([])), "binding", "task", &mut j, &mut Stop).unwrap();
    assert_eq!(result.status, "completed");
    assert_eq!(result.actions, 0);
    assert_eq!(result.state["seen"], "start");
}
struct Work;
impl Application for Work {
    fn on_signal(&mut self, _: &Value, _: &Value) -> Result<Transition> {
        Ok(Transition {
            state: json!({}),
            effect: Effect::Execute {
                capability: "work".into(),
                input: json!({}),
            },
        })
    }
}
#[test]
fn denial_is_persisted_without_launch() {
    let dir = tempfile::tempdir().unwrap();
    let mut j = Journal::open(dir.path()).unwrap();
    let result = run(&manifest(json!([])), "binding", "task", &mut j, &mut Work).unwrap();
    assert_eq!(result.status, "denied");
    assert_eq!(result.actions, 0);
}
#[test]
fn action_budget_stops_an_application_that_keeps_requesting_work() {
    let dir = tempfile::tempdir().unwrap();
    let mut j = Journal::open(dir.path()).unwrap();
    let result = run(
        &manifest(json!(["work"])),
        "binding",
        "task",
        &mut j,
        &mut Work,
    )
    .unwrap();
    assert_eq!(result.status, "budget_hit");
    assert_eq!(result.actions, 1);
}
#[test]
fn restart_never_replays_pending_action_or_accepts_changed_binding() {
    let dir = tempfile::tempdir().unwrap();
    {
        let j = Journal::open(dir.path()).unwrap();
        j.create_run("task", "binding", &json!({})).unwrap();
        let s = j.next_signal("task").unwrap().unwrap();
        j.transition(
            "task",
            &json!({}),
            s.id,
            Some(&json!({"kind":"execute"})),
            "running",
        )
        .unwrap();
    }
    let mut j = Journal::open(dir.path()).unwrap();
    assert!(run(
        &manifest(json!(["work"])),
        "changed",
        "task",
        &mut j,
        &mut Work
    )
    .is_err());
    let result = run(
        &manifest(json!(["work"])),
        "binding",
        "task",
        &mut j,
        &mut Work,
    )
    .unwrap();
    assert_eq!(result.status, "uncertain");
    assert_eq!(result.actions, 1);
}
