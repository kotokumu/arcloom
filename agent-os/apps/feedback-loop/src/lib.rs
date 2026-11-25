#![cfg(any(target_arch = "wasm32", test))]

use serde_json::{json, Value};

fn required_string<'a>(value: &'a Value, key: &str) -> Result<&'a str, String> {
    value
        .get(key)
        .and_then(Value::as_str)
        .filter(|s| !s.is_empty())
        .ok_or_else(|| format!("missing or invalid {key}"))
}

fn transition(mut state: Value, signal: Value) -> Result<Value, String> {
    let kind = required_string(&signal, "kind")?;
    let payload = signal.get("payload").ok_or("missing payload")?;
    if kind == "start" {
        if !state.is_null() {
            return Err("start requires empty state".into());
        }
        required_string(payload, "execution")?;
        required_string(payload, "feedback")?;
        if !payload.get("input").is_some_and(Value::is_object) || payload.get("criteria").is_none()
        {
            return Err("input object and criteria required".into());
        }
        return Ok(
            json!({"state":{"intent":payload,"phase":"executing"},"effect":{
                "kind":"execute","capability":payload["execution"],"input":payload["input"]
            }}),
        );
    }
    let phase = required_string(&state, "phase")?.to_owned();
    let intent = state.get("intent").ok_or("missing saved intent")?.clone();
    required_string(&intent, "execution")?;
    required_string(&intent, "feedback")?;
    if kind == "driver_error" && (phase == "executing" || phase == "evaluating") {
        state["phase"] = json!("failed");
        state["error"] = payload.clone();
        return Ok(json!({"state":state,"effect":{"kind":"wait"}}));
    }
    match (phase.as_str(), kind) {
        ("executing", received) if Some(received) == intent["execution"].as_str() => {
            state["phase"] = json!("evaluating");
            state["result"] = payload.clone();
            Ok(
                json!({"state":state,"effect":{"kind":"evaluate","capability":intent["feedback"],
                "input":{"output":payload,"criteria":intent["criteria"]}}}),
            )
        }
        ("evaluating", received) if Some(received) == intent["feedback"].as_str() => {
            let accepted = payload
                .get("accepted")
                .and_then(Value::as_bool)
                .ok_or("feedback must contain accepted boolean")?;
            state["feedback"] = payload.clone();
            if accepted {
                state["phase"] = json!("complete");
                Ok(json!({"state":state,"effect":{"kind":"complete"}}))
            } else {
                let mut input = intent
                    .get("input")
                    .filter(|v| v.is_object())
                    .ok_or("saved input must be object")?
                    .clone();
                input["feedback"] = payload.clone();
                state["phase"] = json!("executing");
                Ok(
                    json!({"state":state,"effect":{"kind":"execute","capability":intent["execution"],"input":input}}),
                )
            }
        }
        _ => Err("unexpected signal for application phase".into()),
    }
}

#[cfg(target_arch = "wasm32")]
mod component {
    wit_bindgen::generate!({ path: "../../wit", world: "business-app" });
    use arcloom::os::syscalls::{self, Outcome, SyscallError};
    use serde_json::{json, Value};
    struct App;
    impl Guest for App {
        fn on_signal(state: String, event: Signal) -> Result<Transition, String> {
            let state: Value = serde_json::from_str(&state).map_err(|e| e.to_string())?;
            let mut payload: Value =
                serde_json::from_str(&event.payload).map_err(|e| e.to_string())?;
            let intent = if event.kind == "start" {
                &payload
            } else {
                &state["intent"]
            };
            let mut knowledge = None;
            if event.kind == "start" && intent["read_knowledge"].as_bool() == Some(true) {
                let view = match syscalls::knowledge_query() {
                    Ok(view) => view,
                    Err(SyscallError::Denied) => {
                        return Ok(Transition {
                            state: state.to_string(),
                            effect: Outcome::Denied,
                        })
                    }
                    Err(e) => return Err(format!("knowledge unavailable: {e:?}")),
                };
                knowledge = Some(
                    json!({"confirmed":view.confirmed.iter().map(|n|n.id.clone()).collect::<Vec<_>>(),"provisional":view.provisional.iter().map(|n|n.id.clone()).collect::<Vec<_>>()}),
                );
            }
            let mut artifact = None;
            if intent["read_artifact"].as_bool() == Some(true)
                && Some(event.kind.as_str()) == intent["execution"].as_str()
            {
                let hash = event
                    .artifact
                    .as_ref()
                    .ok_or("execution missing artifact")?;
                let bytes = match syscalls::artifact_get(hash) {
                    Ok(bytes) => bytes,
                    Err(SyscallError::Denied) => {
                        return Ok(Transition {
                            state: state.to_string(),
                            effect: Outcome::Denied,
                        })
                    }
                    Err(e) => return Err(format!("artifact unavailable: {e:?}")),
                };
                payload = serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
                artifact = Some(hash.clone());
            }
            let mut result =
                super::transition(state, json!({"kind":event.kind,"payload":payload}))?;
            if let Some(view) = knowledge {
                result["state"]["knowledge_view"] = view;
            }
            if let Some(hash) = artifact {
                result["state"]["read_artifact"] = json!(hash);
            }
            let effect = match result["effect"]["kind"].as_str() {
                Some("wait") => Outcome::Wait,
                Some("complete") => Outcome::Complete,
                Some("execute" | "evaluate") => {
                    let capability = super::required_string(&result["effect"], "capability")?;
                    let lease = match syscalls::lease_acquire(capability) {
                        Ok(lease) => lease,
                        Err(SyscallError::Denied) => {
                            return Ok(Transition {
                                state: result["state"].to_string(),
                                effect: Outcome::Denied,
                            })
                        }
                        Err(e) => return Err(format!("lease unavailable: {e:?}")),
                    };
                    match syscalls::dispatch(&lease, &result["effect"]["input"].to_string()) {
                        Ok(ticket) => Outcome::Dispatch(ticket),
                        Err(SyscallError::Denied) => Outcome::Denied,
                        Err(e) => return Err(format!("dispatch rejected: {e:?}")),
                    }
                }
                _ => return Err("invalid local application effect".into()),
            };
            Ok(Transition {
                state: result["state"].to_string(),
                effect,
            })
        }
    }
    export!(App);
}

#[cfg(test)]
mod tests {
    use super::*;

    fn intent() -> Value {
        json!({"execution":"text.transform","feedback":"text.check", "input":{"text":"hello", "operation":"upper"},"criteria":{"expected":"HELLO"}})
    }

    #[test]
    fn application_owns_execution_then_feedback_then_completion() {
        let started = transition(Value::Null, json!({"kind":"start","payload":intent()})).unwrap();
        assert_eq!(
            started["effect"],
            json!({"kind":"execute","capability":"text.transform","input":{"text":"hello","operation":"upper"}})
        );
        let executed = transition(
            started["state"].clone(),
            json!({"kind":"text.transform","payload":{"text":"HELLO"}}),
        )
        .unwrap();
        assert_eq!(
            executed["effect"],
            json!({"kind":"evaluate","capability":"text.check","input":{"output":{"text":"HELLO"},"criteria":{"expected":"HELLO"}}})
        );
        let accepted = transition(
            executed["state"].clone(),
            json!({"kind":"text.check","payload":{"accepted":true}}),
        )
        .unwrap();
        assert_eq!(accepted["effect"]["kind"], "complete");
        assert_eq!(accepted["state"]["result"]["text"], "HELLO");
    }

    #[test]
    fn capability_names_come_from_intent_instead_of_provider_constants() {
        let mut selected = intent();
        selected["execution"] = json!("custom.command");
        selected["feedback"] = json!("external.score");
        let started = transition(Value::Null, json!({"kind":"start","payload":selected})).unwrap();
        assert_eq!(started["effect"]["capability"], "custom.command");
        let executed = transition(
            started["state"].clone(),
            json!({"kind":"custom.command","payload":{"text":"HELLO"}}),
        )
        .unwrap();
        assert_eq!(executed["effect"]["capability"], "external.score");
        let finished = transition(
            executed["state"].clone(),
            json!({"kind":"external.score","payload":{"accepted":true}}),
        )
        .unwrap();
        assert_eq!(finished["effect"]["kind"], "complete");
    }

    #[test]
    fn rejected_feedback_is_supplied_to_another_execution() {
        let started = transition(Value::Null, json!({"kind":"start","payload":intent()})).unwrap();
        let executed = transition(
            started["state"].clone(),
            json!({"kind":"text.transform","payload":{"text":"wrong"}}),
        )
        .unwrap();
        let retried = transition(
            executed["state"].clone(),
            json!({"kind":"text.check","payload":{"accepted":false,"reason":"wrong text"}}),
        )
        .unwrap();
        assert_eq!(retried["effect"]["kind"], "execute");
        assert_eq!(
            retried["effect"]["input"]["feedback"]["reason"],
            "wrong text"
        );
    }

    #[test]
    fn malformed_or_out_of_order_signals_are_rejected() {
        assert!(transition(Value::Null, json!({"kind":"start","payload":{}})).is_err());
        let started = transition(Value::Null, json!({"kind":"start","payload":intent()})).unwrap();
        assert!(transition(
            started["state"].clone(),
            json!({"kind":"text.check","payload":{"accepted":true}})
        )
        .is_err());
        assert!(transition(
            started["state"].clone(),
            json!({"kind":"unknown","payload":null})
        )
        .is_err());
    }

    #[test]
    fn a_driver_failure_does_not_claim_success_or_retry_itself() {
        let started = transition(Value::Null, json!({"kind":"start","payload":intent()})).unwrap();
        let stopped = transition(
            started["state"].clone(),
            json!({"kind":"driver_error","payload":{"message":"failed"}}),
        )
        .unwrap();
        assert_eq!(stopped["effect"]["kind"], "wait");
        assert_eq!(stopped["state"]["phase"], "failed");
    }
}
