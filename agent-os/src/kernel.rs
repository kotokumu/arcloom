//! Effect authorization and durable dispatch; no business success policy.
use crate::{
    application::Application,
    drivers::Driver,
    journal::{Journal, Run},
    model::{Effect, Manifest},
};
use anyhow::{ensure, Result};
use serde_json::json;
use std::time::{SystemTime, UNIX_EPOCH};

pub fn now() -> Result<i64> {
    Ok(SystemTime::now()
        .duration_since(UNIX_EPOCH)?
        .as_secs()
        .try_into()?)
}

pub fn run(
    m: &Manifest,
    binding: &str,
    id: &str,
    j: &mut Journal,
    app: &mut dyn Application,
) -> Result<Run> {
    m.validate()?;
    ensure!(!id.trim().is_empty(), "run id must not be empty");
    if j.run(id).is_err() {
        j.create_run(id, binding, &m.intent)?;
    }
    ensure!(
        j.run(id)?.binding == binding,
        "run binding changed; use a new run id"
    );
    j.recover(id)?;
    loop {
        let current = j.run(id)?;
        if matches!(
            current.status.as_str(),
            "completed" | "uncertain" | "budget_hit" | "denied" | "failed"
        ) {
            return Ok(current);
        }
        let Some(signal) = j.next_signal(id)? else {
            return Ok(current);
        };
        let context = j.context(now()?)?;
        let mut input_signal = serde_json::to_value(&signal)?;
        input_signal["context"] = context.clone();
        let transition = app.on_signal(&current.state, &input_signal)?;
        let (capability, input, kind, completion) = match &transition.effect {
            Effect::Wait => {
                j.transition(id, &transition.state, signal.id, None, "waiting")?;
                return j.run(id);
            }
            Effect::Complete => {
                j.transition(id, &transition.state, signal.id, None, "completed")?;
                return j.run(id);
            }
            Effect::Execute { capability, input } => (capability, input, "execution", "execution"),
            Effect::Evaluate { capability, input } => (capability, input, "feedback", "feedback"),
        };
        let Ok(provider) = m.authorize(capability, kind) else {
            j.transition(id, &current.state, signal.id, None, "denied")?;
            return j.run(id);
        };
        if current.actions >= m.limits.max_actions {
            j.transition(id, &current.state, signal.id, None, "budget_hit")?;
            return j.run(id);
        }
        let action = j
            .transition(
                id,
                &transition.state,
                signal.id,
                Some(&serde_json::to_value(&transition.effect)?),
                "running",
            )?
            .expect("effect transition returns an action");
        let request = json!({"input":input,"intent":current.intent,"context":context,"action":action,"generation":action});
        match provider.command.invoke(&request) {
            Ok(output) => {
                j.complete_action(id, action, completion, &output)?;
            }
            Err(_) => {
                // Timeout, exit and transport failures cannot establish absence of an effect.
                // Keep the attempt uncertain instead of automatically retrying it.
                j.recover(id)?;
                return j.run(id);
            }
        }
    }
}
