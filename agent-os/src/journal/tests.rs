use super::*;
use anyhow::Result;
use serde_json::{json, Value};
use tempfile::tempdir;

#[test]
fn pending_effect_is_uncertain_after_restart_and_never_replayed() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("task", "binding", &json!({"goal": "sample"}))?;
    let signal = db.next_signal("task")?.unwrap();
    let action = db
        .transition(
            "task",
            &json!({"step": 1}),
            signal.id,
            Some(&json!({"driver": "worker"})),
            "running",
        )?
        .unwrap();
    drop(db);
    let db = Journal::open(dir.path())?;
    assert!(db.recover("task")?);
    assert_eq!(db.run("task")?.status, "uncertain");
    assert_eq!(db.run("task")?.actions, 1);
    assert!(!db.recover("task")?);
    assert!(db
        .complete_action("task", action, "feedback", &json!("late"))
        .is_err());
    assert_eq!(db.signals("task")?.last().unwrap().kind, "stall");
    Ok(())
}

#[test]
fn completed_action_survives_reopen_without_duplicate_delivery() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("task", "binding", &json!({}))?;
    let input = db.next_signal("task")?.unwrap();
    let action = db
        .transition("task", &json!(1), input.id, Some(&json!({})), "running")?
        .unwrap();
    db.complete_action("task", action, "feedback", &json!({"answer": 42}))?;
    assert!(db
        .complete_action("task", action, "feedback", &json!({}))
        .is_err());
    drop(db);
    let db = Journal::open(dir.path())?;
    assert!(!db.recover("task")?);
    let response = db.next_signal("task")?.unwrap();
    assert_eq!(response.action, Some(action));
    assert_eq!(response.payload["answer"], 42);
    assert_eq!(
        serde_json::from_slice::<Value>(&db.get_artifact(response.artifact.as_deref().unwrap())?)?,
        response.payload
    );
    db.transition("task", &json!(2), response.id, None, "complete")?;
    assert!(db.next_signal("task")?.is_none());
    assert!(db
        .transition("task", &json!(3), response.id, None, "complete")
        .is_err());
    assert_eq!(db.run("task")?.state, json!(2));
    Ok(())
}

#[test]
fn failed_transition_is_atomic_and_cannot_ack_another_run() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("a", "a", &json!({}))?;
    db.create_run("b", "b", &json!({}))?;
    let input = db.next_signal("b")?.unwrap();
    assert!(db
        .transition(
            "a",
            &json!("changed"),
            input.id,
            Some(&json!({})),
            "running"
        )
        .is_err());
    assert_eq!(db.run("a")?.actions, 0);
    assert_eq!(db.run("a")?.state, Value::Null);
    assert!(db.next_signal("a")?.is_some());
    Ok(())
}

#[test]
fn artifacts_are_content_addressed_and_corruption_is_detected() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    let hash = db.put_artifact(b"original")?;
    assert_eq!(db.put_artifact(b"original")?, hash);
    assert_eq!(db.get_artifact(&hash)?, b"original");
    assert!(db.get_artifact("../journal.sqlite").is_err());
    std::fs::write(dir.path().join("cas").join(&hash), b"tampered")?;
    assert!(db.get_artifact(&hash).is_err());
    assert!(db.put_artifact(b"original").is_err());
    Ok(())
}

#[test]
fn knowledge_context_separates_authority_and_excludes_expiry() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    let source = db.put_artifact(b"evidence")?;
    db.add_knowledge(&json!({"id":"policy","kind":"decision","status":"stable","trust":"human","sources":[source],"text":"rule"}))?;
    db.add_knowledge(&json!({"id":"note","kind":"hypothesis","status":"draft","trust":"unverified","stale_after":20,"sources":[source],"text":"maybe"}))?;
    assert!(db
        .add_knowledge(
            &json!({"id":"bad","kind":"finding","status":"draft","trust":"unverified","sources":[]})
        )
        .is_err());
    assert!(db.add_knowledge(&json!({"id":"bad-source","kind":"finding","status":"draft","trust":"unverified","stale_after":20,"sources":["missing"]})).is_err());
    let before = db.context(19)?;
    assert_eq!(before["confirmed"].as_array().unwrap().len(), 1);
    assert_eq!(before["provisional"].as_array().unwrap().len(), 1);
    assert_eq!(db.context(20)?["provisional"], json!([]));
    assert_eq!(db.context(20)?["confirmed"].as_array().unwrap().len(), 1);
    Ok(())
}

#[test]
fn leases_fence_expired_holders_and_survive_reopen() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    let first = db.lease_acquire("workspace", "a", 10, 5)?;
    assert!(db.lease_acquire("workspace", "b", 14, 5).is_err());
    drop(db);
    let db = Journal::open(dir.path())?;
    let second = db.lease_acquire("workspace", "b", 15, 5)?;
    assert!(second > first);
    assert!(db.lease_release("workspace", "a", first).is_err());
    db.lease_release("workspace", "b", second)?;
    let third = db.lease_acquire("workspace", "a", 16, 5)?;
    assert!(third > second);
    assert!(db.lease_acquire("invalid", "a", 0, 0).is_err());
    Ok(())
}
#[test]
fn mailbox_coalescing_preserves_interleaved_queue_and_consumer_independence() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("task", "binding", &json!({}))?;
    let first = db.append_signal("task", "timer", &json!({"target":"x","n":1}))?;
    let queue = db.append_signal("task", "human_steer", &json!({"text":"keep this"}))?;
    let latest = db.append_signal("task", "timer", &json!({"target":"x","n":2}))?;
    db.register_mailbox(
        "app",
        "task",
        &["timer".into(), "human_steer".into()],
        &["timer".into()],
    )?;
    db.register_mailbox(
        "audit",
        "task",
        &["timer".into(), "human_steer".into()],
        &[],
    )?;
    assert_eq!(db.mailbox_next("app")?.unwrap().id, latest);
    db.mailbox_ack("app", latest)?;
    drop(db);
    let db = Journal::open(dir.path())?;
    assert_eq!(db.mailbox_next("app")?.unwrap().id, queue);
    assert_eq!(db.mailbox_next("audit")?.unwrap().id, first);
    db.mailbox_ack("app", queue)?;
    assert!(db.mailbox_next("app")?.is_none());
    assert!(db.mailbox_ack("app", latest).is_err());
    assert!(db
        .register_mailbox("app", "task", &["other".into()], &[])
        .is_err());
    Ok(())
}
#[test]
fn failed_cas_write_does_not_commit_action_completion() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("task", "binding", &json!({}))?;
    let input = db.next_signal("task")?.unwrap();
    let action = db
        .transition("task", &json!(1), input.id, Some(&json!({})), "running")?
        .unwrap();
    let payload = json!({"result": "important"});
    let hash = db.put_artifact(&serde_json::to_vec(&payload)?)?;
    std::fs::write(dir.path().join("cas").join(hash), b"corruption")?;
    assert!(db
        .complete_action("task", action, "feedback", &payload)
        .is_err());
    assert_eq!(db.signals("task")?.len(), 1);
    assert!(db.recover("task")?);
    Ok(())
}
#[test]
fn coalescing_ack_keeps_notifications_arriving_during_delivery() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("task", "binding", &json!({}))?;
    db.register_mailbox("app", "task", &["timer".into()], &["timer".into()])?;
    let first = db.append_signal("task", "timer", &json!({"target":"x"}))?;
    assert_eq!(db.mailbox_next("app")?.unwrap().id, first);
    let newer = db.append_signal("task", "timer", &json!({"target":"x"}))?;
    db.mailbox_ack("app", first)?;
    assert_eq!(db.mailbox_next("app")?.unwrap().id, newer);
    Ok(())
}

#[test]
fn schema_one_migration_preserves_existing_runs() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("task", "binding", &json!({"goal":"retained"}))?;
    db.connection
            .execute_batch("ALTER TABLE actions DROP COLUMN artifact; ALTER TABLE actions DROP COLUMN resource; ALTER TABLE actions DROP COLUMN lease_token; PRAGMA user_version=1;")?;
    drop(db);
    let db = Journal::open(dir.path())?;
    assert_eq!(db.run("task")?.intent["goal"], "retained");
    let input = db.next_signal("task")?.unwrap();
    let action = db
        .transition("task", &json!(1), input.id, Some(&json!({})), "running")?
        .unwrap();
    db.complete_action("task", action, "feedback", &json!({"ok":true}))?;
    assert!(db.next_signal("task")?.unwrap().artifact.is_some());
    Ok(())
}
#[test]
fn concurrent_transitions_cannot_dispatch_the_same_input_twice() -> Result<()> {
    let dir = tempdir()?;
    let first = Journal::open(dir.path())?;
    first.create_run("task", "binding", &json!({}))?;
    let signal = first.next_signal("task")?.unwrap().id;
    drop(first);
    let first_path = dir.path().to_path_buf();
    let second_path = first_path.clone();
    let barrier = std::sync::Arc::new(std::sync::Barrier::new(2));
    let other_barrier = barrier.clone();
    let a = std::thread::spawn(move || {
        let first = Journal::open(&first_path)?;
        barrier.wait();
        first.transition(
            "task",
            &json!(1),
            signal,
            Some(&json!({"which":"a"})),
            "running",
        )
    });
    let b = std::thread::spawn(move || {
        let second = Journal::open(&second_path)?;
        other_barrier.wait();
        second.transition(
            "task",
            &json!(2),
            signal,
            Some(&json!({"which":"b"})),
            "running",
        )
    });
    let results = [a.join().unwrap(), b.join().unwrap()];
    assert_eq!(results.iter().filter(|result| result.is_ok()).count(), 1);
    let db = Journal::open(dir.path())?;
    assert_eq!(db.run("task")?.actions, 1);
    assert!(db.recover("task")?);
    Ok(())
}

#[test]
fn task_mailbox_commits_coalescing_and_state_together() -> Result<()> {
    use crate::storage::{Mailbox, RunStore};
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.open_run("t", "binding", &json!({}))?;
    db.prepare_mailbox("t", &["timer".into()])?;
    let start = db.receive("t")?.unwrap();
    db.commit("t", &json!(0), start.id, "waiting")?;
    db.append_signal("t", "timer", &json!({"target":"x","n":1}))?;
    let queue = db.append_signal("t", "human_steer", &json!({"text":"keep"}))?;
    let latest = db.append_signal("t", "timer", &json!({"target":"x","n":2}))?;
    assert_eq!(db.receive("t")?.unwrap().id, latest);
    db.commit("t", &json!(1), latest, "waiting")?;
    drop(db);
    let db = Journal::open(dir.path())?;
    assert_eq!(db.receive("t")?.unwrap().id, queue);
    assert_eq!(db.run("t")?.state, json!(1));
    db.commit("t", &json!(2), queue, "waiting")?;
    assert!(db.receive("t")?.is_none());
    Ok(())
}

#[test]
fn reservation_contention_preserves_input_and_completion_releases_exact_lease() -> Result<()> {
    use crate::storage::{LeaseManager, Mailbox, Reservation, RunStore};
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    for run in ["a", "b"] {
        db.open_run(run, run, &json!({}))?;
    }
    let a = db.receive("a")?.unwrap().id;
    let b = db.receive("b")?.unwrap().id;
    let state = json!(1);
    let effect = json!({"capability":"feedback","input":"opaque"});
    let first = db
        .reserve(Reservation {
            run: "a",
            state: &state,
            signal: a,
            effect: &effect,
            resource: "resource",
            now: 10,
            ttl: 5,
        })?
        .unwrap();
    assert!(db
        .reserve(Reservation {
            run: "b",
            state: &state,
            signal: b,
            effect: &effect,
            resource: "resource",
            now: 11,
            ttl: 5
        })?
        .is_none());
    assert_eq!(db.receive("b")?.unwrap().id, b);
    assert_eq!(db.run("b")?.actions, 0);
    assert_eq!(db.run("b")?.state, Value::Null);
    db.complete_action("a", first.action, "feedback", &json!({"ok":true}))?;
    let next = db
        .reserve(Reservation {
            run: "b",
            state: &state,
            signal: b,
            effect: &effect,
            resource: "resource",
            now: 12,
            ttl: 5,
        })?
        .unwrap();
    assert!(next.token > first.token);
    assert!(db.lease_release("resource", "a", first.token).is_err());
    Ok(())
}

#[test]
fn uncertain_reservation_stays_protected_past_expiry_until_explicit_reconciliation() -> Result<()> {
    use crate::storage::{LeaseManager, Mailbox, Reservation, RunStore};
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    for run in ["a", "b"] {
        db.open_run(run, run, &json!({}))?;
    }
    let a = db.receive("a")?.unwrap().id;
    let b = db.receive("b")?.unwrap().id;
    let state = json!(1);
    let effect = json!({"capability":"feedback"});
    let first = db
        .reserve(Reservation {
            run: "a",
            state: &state,
            signal: a,
            effect: &effect,
            resource: "r",
            now: 1,
            ttl: 1,
        })?
        .unwrap();
    drop(db);
    let db = Journal::open(dir.path())?;
    assert!(db.recover("a")?);
    assert!(db
        .reserve(Reservation {
            run: "b",
            state: &state,
            signal: b,
            effect: &effect,
            resource: "r",
            now: 100,
            ttl: 1
        })?
        .is_none());
    assert!(db.lease_release("r", "a", first.token).is_err());
    db.reconcile_lease("r", "a", first.token)?;
    assert!(db
        .reserve(Reservation {
            run: "b",
            state: &state,
            signal: b,
            effect: &effect,
            resource: "r",
            now: 100,
            ttl: 1
        })?
        .is_some());
    Ok(())
}

#[test]
fn open_run_rejects_changed_intent_without_resetting_existing_state() -> Result<()> {
    use crate::storage::RunStore;
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.open_run("t", "binding", &json!({"goal":"one"}))?;
    assert!(db.open_run("t", "other", &json!({"goal":"one"})).is_err());
    assert!(db.open_run("t", "binding", &json!({"goal":"two"})).is_err());
    assert_eq!(db.signals("t")?.len(), 1);
    Ok(())
}

#[test]
fn completion_requires_exact_capability_and_live_fencing_token() -> Result<()> {
    use crate::storage::{LeaseManager, Mailbox, Reservation, RunStore};
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.open_run("t", "binding", &json!({}))?;
    let signal = db.receive("t")?.unwrap().id;
    let receipt = db
        .reserve(Reservation {
            run: "t",
            state: &json!(1),
            signal,
            effect: &json!({"capability":"expected"}),
            resource: "r",
            now: 1,
            ttl: 10,
        })?
        .unwrap();
    assert!(db
        .complete_action("t", receipt.action, "wrong", &json!({}))
        .is_err());
    assert_eq!(db.signals("t")?.len(), 1);
    db.connection
        .execute("UPDATE leases SET token=token+1 WHERE resource='r'", [])?;
    assert!(db
        .complete_action("t", receipt.action, "expected", &json!({}))
        .is_err());
    assert_eq!(db.signals("t")?.len(), 1);
    assert!(db.recover("t")?);
    Ok(())
}

#[test]
fn operator_cannot_reconcile_an_expired_but_still_pending_action() -> Result<()> {
    use crate::storage::{LeaseManager, Mailbox, Reservation, RunStore};
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    for run in ["a", "b"] {
        db.open_run(run, run, &json!({}))?;
    }
    let a = db.receive("a")?.unwrap().id;
    let b = db.receive("b")?.unwrap().id;
    let effect = json!({"capability":"work"});
    let state = json!(1);
    let receipt = db
        .reserve(Reservation {
            run: "a",
            state: &state,
            signal: a,
            effect: &effect,
            resource: "r",
            now: 1,
            ttl: 1,
        })?
        .unwrap();
    assert!(db
        .reserve(Reservation {
            run: "b",
            state: &state,
            signal: b,
            effect: &effect,
            resource: "r",
            now: 10,
            ttl: 1
        })?
        .is_none());
    assert!(db.reconcile_lease("r", "a", receipt.token).is_err());
    db.recover("a")?;
    db.reconcile_lease("r", "a", receipt.token)?;
    Ok(())
}

#[test]
fn schema_two_upgrade_keeps_completed_evidence_and_mailbox_position() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("task", "binding", &json!({"goal":"keep"}))?;
    let first = db.next_signal("task")?.unwrap();
    let action = db
        .transition(
            "task",
            &json!({"step":1}),
            first.id,
            Some(&json!({})),
            "running",
        )?
        .unwrap();
    db.complete_action("task", action, "feedback", &json!({"result":"retained"}))?;
    let artifact = db.next_signal("task")?.unwrap().artifact.unwrap();
    db.connection.execute_batch("ALTER TABLE actions DROP COLUMN resource; ALTER TABLE actions DROP COLUMN lease_token; PRAGMA user_version=2;")?;
    drop(db);
    let db = Journal::open(dir.path())?;
    assert_eq!(db.run("task")?.actions, 1);
    assert_eq!(db.run("task")?.state, json!({"step":1}));
    assert_eq!(
        db.next_signal("task")?.unwrap().artifact.as_deref(),
        Some(artifact.as_str())
    );
    assert_eq!(
        serde_json::from_slice::<Value>(&db.get_artifact(&artifact)?)?,
        json!({"result":"retained"})
    );
    assert!(!db.recover("task")?);
    Ok(())
}

#[test]
fn future_schema_is_rejected_without_resetting_history() -> Result<()> {
    let dir = tempdir()?;
    let db = Journal::open(dir.path())?;
    db.create_run("task", "binding", &json!({"goal":"preserve"}))?;
    db.connection.pragma_update(None, "user_version", 99)?;
    drop(db);
    assert!(Journal::open(dir.path()).is_err());
    let connection = rusqlite::Connection::open(dir.path().join("journal.sqlite"))?;
    let version: i64 = connection.pragma_query_value(None, "user_version", |row| row.get(0))?;
    assert_eq!(version, 99);
    let count: i64 =
        connection.query_row("SELECT COUNT(*) FROM runs WHERE id='task'", [], |row| {
            row.get(0)
        })?;
    assert_eq!(count, 1);
    Ok(())
}

#[test]
fn completion_and_knowledge_use_the_same_substituted_evidence_store() -> Result<()> {
    use crate::storage::{ArtifactStore, LeaseManager, Mailbox, Reservation, RunStore};
    use sha2::{Digest, Sha256};
    use std::{
        cell::{Cell, RefCell},
        collections::BTreeMap,
        rc::Rc,
    };
    #[derive(Default)]
    struct MemoryEvidence {
        objects: RefCell<BTreeMap<String, Vec<u8>>>,
        writes: Cell<usize>,
        reads: Cell<usize>,
    }
    impl ArtifactStore for MemoryEvidence {
        fn get_artifact(&self, hash: &str) -> Result<Vec<u8>> {
            self.reads.set(self.reads.get() + 1);
            self.objects
                .borrow()
                .get(hash)
                .cloned()
                .ok_or_else(|| anyhow::anyhow!("unavailable evidence"))
        }
    }
    impl EvidenceStore for MemoryEvidence {
        fn put_artifact(&self, bytes: &[u8]) -> Result<String> {
            self.writes.set(self.writes.get() + 1);
            let hash = format!("{:x}", Sha256::digest(bytes));
            self.objects
                .borrow_mut()
                .entry(hash.clone())
                .or_insert_with(|| bytes.to_vec());
            Ok(hash)
        }
    }
    let dir = tempdir()?;
    let artifacts = Rc::new(MemoryEvidence::default());
    let db = Journal::open_with_artifacts(dir.path(), artifacts.clone())?;
    db.open_run("task", "binding", &json!({}))?;
    let input = db.receive("task")?.unwrap();
    let receipt = db
        .reserve(Reservation {
            run: "task",
            state: &json!(1),
            signal: input.id,
            effect: &json!({"capability":"work"}),
            resource: "r",
            now: 1,
            ttl: 10,
        })?
        .unwrap();
    let result = json!({"answer":42});
    db.complete_action("task", receipt.action, "work", &result)?;
    let signal = db.receive("task")?.unwrap();
    let hash = signal.artifact.unwrap();
    assert_eq!(artifacts.writes.get(), 1);
    assert_eq!(
        artifacts.objects.borrow().get(&hash),
        Some(&serde_json::to_vec(&result)?)
    );
    assert_eq!(
        serde_json::from_slice::<Value>(&ArtifactStore::get_artifact(&db, &hash)?)?,
        result
    );
    db.add_knowledge(&json!({"id":"finding","kind":"finding","status":"draft","trust":"unverified","stale_after":20,"sources":[hash]}))?;
    assert_eq!(db.context(10)?["provisional"].as_array().unwrap().len(), 1);
    assert!(artifacts.reads.get() >= 3);
    assert!(!dir.path().join("cas").exists());
    Ok(())
}
