#![cfg(any(target_arch = "wasm32", test))]

use serde_json::{json, Value};

fn transition(state: Value, signal: Value) -> Result<Value, String> {
    if !state.is_null() || signal.get("kind").and_then(Value::as_str) != Some("start") {
        return Err("expected initial start signal".into());
    }
    let intent = signal.get("payload").ok_or("missing intent")?;
    Ok(json!({"state":{"observed_intent":intent},"effect":{"kind":"complete"}}))
}

#[cfg(target_arch = "wasm32")]
mod component {
    wit_bindgen::generate!({ path: "../../wit", world: "business-app" });
    use arcloom::os::syscalls::Outcome;
    struct App;
    impl Guest for App {
        fn on_signal(state: String, event: Signal) -> Result<Transition, String> {
            let state = serde_json::from_str(&state).map_err(|e| e.to_string())?;
            let payload: serde_json::Value =
                serde_json::from_str(&event.payload).map_err(|e| e.to_string())?;
            let value = super::transition(
                state,
                serde_json::json!({"kind":event.kind,"payload":payload}),
            )?;
            Ok(Transition {
                state: value["state"].to_string(),
                effect: Outcome::Complete,
            })
        }
    }
    export!(App);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn completes_without_execution_or_evaluation() {
        let result = transition(
            Value::Null,
            json!({"kind":"start","payload":{"goal":"observe"}}),
        )
        .unwrap();
        assert_eq!(
            result,
            json!({"state":{"observed_intent":{"goal":"observe"}},"effect":{"kind":"complete"}})
        );
    }

    #[test]
    fn rejects_malformed_or_repeated_start() {
        assert!(transition(Value::Null, json!({"kind":"start"})).is_err());
        assert!(transition(
            json!({"observed_intent":{}}),
            json!({"kind":"start","payload":{}})
        )
        .is_err());
        assert!(transition(Value::Null, json!({"kind":"execution","payload":{}})).is_err());
    }
}
