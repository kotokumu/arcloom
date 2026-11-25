use arcloom_os::model::Manifest;
use serde_json::json;
fn manifest() -> Manifest {
    serde_json::from_value(json!({
        "version":1,"component":"app.wasm","intent":{"goal":"operator chosen"},
        "requires":["work"],"grants":[],
        "drivers":[{"id":"one","kind":"execution","provides":["work"],"command":{
            "argv":["/usr/bin/python3"],"cwd":"/tmp","env":{},"timeout_ms":1000,"max_output_bytes":4096}}],
        "limits":{"max_actions":3,"fuel":1000000,"memory_bytes":16777216}
    })).unwrap()
}
#[test]
fn no_implicit_grant_or_kind_escalation() {
    let mut m = manifest();
    m.validate().unwrap();
    assert!(m.authorize("work", "execution").is_err());
    m.grants.push("work".into());
    assert!(m.authorize("work", "execution").is_ok());
    assert!(m.authorize("work", "feedback").is_err());
}
#[test]
fn unresolved_and_ambiguous_dependencies_rejected() {
    let mut m = manifest();
    m.requires.push("missing".into());
    assert!(m.validate().is_err());
    let mut m = manifest();
    m.drivers[0].provides.push("work".into());
    assert!(m.validate().is_err());
}
#[test]
fn zero_limits_and_unknown_manifest_fields_rejected() {
    let mut m = manifest();
    m.limits.max_actions = 0;
    assert!(m.validate().is_err());
    let mut v = serde_json::to_value(manifest()).unwrap();
    v["ambient_authority"] = json!(true);
    assert!(serde_json::from_value::<Manifest>(v).is_err());
}
