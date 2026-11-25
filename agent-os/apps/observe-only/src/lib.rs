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
