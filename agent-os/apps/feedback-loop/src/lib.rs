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
        ("executing", "execution") => {
            state["phase"] = json!("evaluating");
            state["result"] = payload.clone();
            Ok(
                json!({"state":state,"effect":{"kind":"evaluate","capability":intent["feedback"],
                "input":{"output":payload,"criteria":intent["criteria"]}}}),
            )
        }
        ("evaluating", "feedback") => {
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
    struct App;
    impl Guest for App {
        fn on_signal(state: String, signal: String) -> String {
            let result = serde_json::from_str(&state)
                .map_err(|e| e.to_string())
                .and_then(|state| {
                    serde_json::from_str(&signal)
                        .map_err(|e| e.to_string())
                        .map(|signal| (state, signal))
                })
                .and_then(|(state, signal)| super::transition(state, signal));
            match result {
                Ok(value) => value.to_string(),
                Err(error) => serde_json::json!({"error":error}).to_string(),
            }
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
            json!({"kind":"execution","payload":{"text":"HELLO"}}),
        )
        .unwrap();
        assert_eq!(
            executed["effect"],
            json!({"kind":"evaluate","capability":"text.check","input":{"output":{"text":"HELLO"},"criteria":{"expected":"HELLO"}}})
        );
        let accepted = transition(
            executed["state"].clone(),
            json!({"kind":"feedback","payload":{"accepted":true}}),
        )
        .unwrap();
        assert_eq!(accepted["effect"]["kind"], "complete");
        assert_eq!(accepted["state"]["result"]["text"], "HELLO");
    }

    #[test]
    fn rejected_feedback_is_supplied_to_another_execution() {
        let started = transition(Value::Null, json!({"kind":"start","payload":intent()})).unwrap();
        let executed = transition(
            started["state"].clone(),
            json!({"kind":"execution","payload":{"text":"wrong"}}),
        )
        .unwrap();
        let retried = transition(
            executed["state"].clone(),
            json!({"kind":"feedback","payload":{"accepted":false,"reason":"wrong text"}}),
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
            json!({"kind":"feedback","payload":{"accepted":true}})
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
