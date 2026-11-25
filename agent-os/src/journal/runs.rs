use super::mailbox::task_consumer;
use super::Journal;
use crate::storage::{Action, Mailbox, Run, RunStore, Signal};
use anyhow::{ensure, Context, Result};
use rusqlite::Transaction;
use rusqlite::{params, OptionalExtension};
use serde_json::{json, Value};

impl Journal {
    pub fn create_run(&self, id: &str, binding: &str, intent: &Value) -> Result<()> {
        ensure!(
            !id.is_empty() && !binding.is_empty(),
            "run ID and binding must not be empty"
        );
        let transaction = self.transaction()?;
        transaction.execute(
            "INSERT INTO runs(id,binding,intent,state,status) VALUES (?1,?2,?3,'null','ready')",
            params![id, binding, intent.to_string()],
        )?;
        transaction.execute(
            "INSERT INTO signals(run,kind,payload) VALUES (?1,'start',?2)",
            params![id, intent.to_string()],
        )?;
        transaction.commit()?;
        Ok(())
    }

    pub fn run(&self, id: &str) -> Result<Run> {
        let (binding, intent, state, status, actions, cursor): (
            String,
            String,
            String,
            String,
            u64,
            i64,
        ) = self
            .connection
            .query_row(
                "SELECT binding,intent,state,status,actions,cursor FROM runs WHERE id=?1",
                [id],
                |row| {
                    Ok((
                        row.get(0)?,
                        row.get(1)?,
                        row.get(2)?,
                        row.get(3)?,
                        row.get(4)?,
                        row.get(5)?,
                    ))
                },
            )
            .with_context(|| format!("unknown run {id}"))?;
        Ok(Run {
            id: id.to_string(),
            binding,
            intent: serde_json::from_str(&intent)?,
            state: serde_json::from_str(&state)?,
            status,
            actions,
            cursor,
        })
    }

    pub fn actions(&self, run: &str) -> Result<Vec<Action>> {
        self.run(run)?;
        let mut statement=self.connection.prepare("SELECT id,status,effect,resource,lease_token,artifact FROM actions WHERE run=?1 ORDER BY id")?;
        let rows = statement.query_map([run], |row| {
            Ok((
                row.get::<_, i64>(0)?,
                row.get::<_, String>(1)?,
                row.get::<_, String>(2)?,
                row.get::<_, Option<String>>(3)?,
                row.get::<_, Option<u64>>(4)?,
                row.get::<_, Option<String>>(5)?,
            ))
        })?;
        rows.map(|row| {
            let (id, status, effect, resource, lease_token, artifact) = row?;
            Ok(Action {
                id,
                run: run.into(),
                generation: id,
                status,
                effect: serde_json::from_str(&effect)?,
                resource,
                lease_token,
                artifact,
            })
        })
        .collect()
    }

    pub fn signals(&self, run: &str) -> Result<Vec<Signal>> {
        self.run(run)?;
        let mut statement = self
            .connection
            .prepare("SELECT s.id,s.kind,s.payload,s.action,a.artifact FROM signals s LEFT JOIN actions a ON a.id=s.action WHERE s.run=?1 ORDER BY s.id")?;
        let rows = statement.query_map([run], |row| {
            Ok((
                row.get::<_, i64>(0)?,
                row.get::<_, String>(1)?,
                row.get::<_, String>(2)?,
                row.get::<_, Option<i64>>(3)?,
                row.get::<_, Option<String>>(4)?,
            ))
        })?;
        rows.map(|row| {
            let (id, kind, payload, action, artifact) = row?;
            Ok(Signal {
                id,
                kind,
                payload: serde_json::from_str(&payload)?,
                action,
                generation: action,
                artifact,
            })
        })
        .collect()
    }

    pub fn next_signal(&self, run: &str) -> Result<Option<Signal>> {
        Mailbox::receive(self, run)
    }

    #[cfg(test)]
    pub fn transition(
        &self,
        run: &str,
        state: &Value,
        signal_id: i64,
        effect: Option<&Value>,
        status: &str,
    ) -> Result<Option<i64>> {
        self.ensure_task_mailbox(run)?;
        let transaction = self.transaction()?;
        let action = self.commit_transition(&transaction, run, state, signal_id, effect, status)?;
        transaction.commit()?;
        Ok(action)
    }

    pub(super) fn commit_transition(
        &self,
        transaction: &Transaction<'_>,
        run: &str,
        state: &Value,
        signal_id: i64,
        effect: Option<&Value>,
        status: &str,
    ) -> Result<Option<i64>> {
        ensure!(!status.is_empty(), "empty run status");
        let cursor = self.ack_mailbox(transaction, &task_consumer(run), signal_id)?;
        let pending: bool = transaction.query_row(
            "SELECT EXISTS(SELECT 1 FROM actions WHERE run=?1 AND status='pending')",
            [run],
            |row| row.get(0),
        )?;
        ensure!(!pending, "an action is already pending");
        let mut action = None;
        if let Some(effect) = effect {
            transaction.execute(
                "INSERT INTO actions(run,effect,status) VALUES (?1,?2,'pending')",
                params![run, effect.to_string()],
            )?;
            action = Some(transaction.last_insert_rowid());
        }
        transaction.execute(
            "UPDATE runs SET state=?1,cursor=?2,status=?3,actions=actions+?4 WHERE id=?5",
            params![
                state.to_string(),
                cursor,
                status,
                i64::from(effect.is_some()),
                run
            ],
        )?;
        if status == "budget_hit" || status == "denied" {
            transaction.execute(
                "INSERT INTO signals(run,kind,payload) VALUES (?1,?2,'{}')",
                params![run, status],
            )?;
        }
        Ok(action)
    }

    pub fn complete_action(
        &self,
        run: &str,
        action: i64,
        kind: &str,
        payload: &Value,
    ) -> Result<()> {
        ensure!(!kind.is_empty(), "signal kind must not be empty");
        let transaction = self.transaction()?;
        let (effect,resource,token):(String,Option<String>,Option<u64>)=transaction.query_row("SELECT effect,resource,lease_token FROM actions WHERE id=?1 AND run=?2 AND status='pending'",params![action,run],|row|Ok((row.get(0)?,row.get(1)?,row.get(2)?))).context("unknown, stale, uncertain or already completed action")?;
        if let Some(resource) = resource {
            let effect: Value = serde_json::from_str(&effect)?;
            ensure!(
                effect["capability"].as_str() == Some(kind),
                "completion capability does not match reserved action"
            );
            let token = token.context("reserved action has no fencing token")?;
            let released=transaction.execute("UPDATE leases SET status='released' WHERE resource=?1 AND token=?2 AND holder=?3 AND status IN ('active','reconciling')",params![resource,token,run])?;
            ensure!(released == 1, "reservation fencing lost before completion");
        }
        let changed = transaction.execute(
            "UPDATE actions SET status='completed' WHERE id=?1 AND run=?2 AND status='pending'",
            params![action, run],
        )?;
        ensure!(
            changed == 1,
            "unknown, stale, uncertain or already completed action"
        );
        // Persist and verify immutable bytes before committing their journal reference.
        // A later SQL failure can leave an unreferenced CAS blob, never a missing referenced blob.
        let artifact = self.put_artifact(&serde_json::to_vec(payload)?)?;
        transaction.execute(
            "UPDATE actions SET artifact=?1 WHERE id=?2",
            params![artifact, action],
        )?;
        transaction.execute(
            "INSERT INTO signals(run,kind,payload,action) VALUES (?1,?2,?3,?4)",
            params![run, kind, payload.to_string(), action],
        )?;
        transaction.execute("UPDATE runs SET status='ready' WHERE id=?1", [run])?;
        transaction.commit()?;
        Ok(())
    }

    pub fn recover(&self, run: &str) -> Result<bool> {
        self.run(run)?;
        let transaction = self.transaction()?;
        let pending: Option<i64> = transaction
            .query_row(
                "SELECT id FROM actions WHERE run=?1 AND status='pending' ORDER BY id LIMIT 1",
                [run],
                |row| row.get(0),
            )
            .optional()?;
        if let Some(action) = pending {
            transaction.execute("UPDATE leases SET status='reconciling' WHERE resource=(SELECT resource FROM actions WHERE id=?1) AND token=(SELECT lease_token FROM actions WHERE id=?1) AND holder=?2",params![action,run])?;
            transaction.execute(
                "UPDATE actions SET status='uncertain' WHERE id=?1",
                [action],
            )?;
            transaction.execute("UPDATE runs SET status='uncertain' WHERE id=?1", [run])?;
            transaction.execute(
                "INSERT INTO signals(run,kind,payload,action) VALUES (?1,'stall',?2,?3)",
                params![
                    run,
                    json!({"reason":"external_outcome_uncertain","replay":false}).to_string(),
                    action
                ],
            )?;
        }
        transaction.commit()?;
        Ok(pending.is_some())
    }

    pub fn append_signal(&self, run: &str, kind: &str, payload: &Value) -> Result<i64> {
        ensure!(!kind.is_empty(), "signal kind must not be empty");
        self.connection.execute(
            "INSERT INTO signals(run,kind,payload) VALUES (?1,?2,?3)",
            params![run, kind, payload.to_string()],
        )?;
        Ok(self.connection.last_insert_rowid())
    }
}

impl RunStore for Journal {
    fn open_run(&self, id: &str, binding: &str, intent: &Value) -> Result<Run> {
        ensure!(
            !id.trim().is_empty() && !binding.is_empty(),
            "run ID and binding must not be empty"
        );
        let transaction = self.transaction()?;
        let existing: Option<(String, String)> = transaction
            .query_row("SELECT binding,intent FROM runs WHERE id=?1", [id], |row| {
                Ok((row.get(0)?, row.get(1)?))
            })
            .optional()?;
        if let Some((saved_binding, saved_intent)) = existing {
            ensure!(
                saved_binding == binding,
                "run binding changed; use a new run ID"
            );
            ensure!(
                serde_json::from_str::<Value>(&saved_intent)? == *intent,
                "run Intent changed; use a new run ID"
            );
        } else {
            transaction.execute(
                "INSERT INTO runs(id,binding,intent,state,status) VALUES (?1,?2,?3,'null','ready')",
                params![id, binding, intent.to_string()],
            )?;
            transaction.execute(
                "INSERT INTO signals(run,kind,payload) VALUES (?1,'start',?2)",
                params![id, intent.to_string()],
            )?;
        }
        transaction.commit()?;
        self.run(id)
    }
    fn run(&self, id: &str) -> Result<Run> {
        Journal::run(self, id)
    }
    fn recover(&self, id: &str) -> Result<bool> {
        Journal::recover(self, id)
    }
    fn complete_action(&self, id: &str, action: i64, kind: &str, payload: &Value) -> Result<()> {
        Journal::complete_action(self, id, action, kind, payload)
    }
}
