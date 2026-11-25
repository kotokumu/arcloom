use arcloom_os::model::Manifest;
use serde_json::json;
fn manifest() -> Manifest {
    serde_json::from_value(json!({
        "version":2,"component":"app.wasm","intent":{"goal":"operator chosen"},
        "requires":["work"],"grants":[],
        "drivers":[{"id":"one","resource":"shared-work","provides":["work"],"requires":[],"adapter":"identity","config":{}}],
        "limits":{"max_actions":3,"fuel":1000000,"memory_bytes":16777216}
    })).unwrap()
}
#[test]
fn root_and_transitive_grants_are_distinct_from_requirements() {
    let mut m = manifest();
    m.drivers.push(serde_json::from_value(json!({"id":"two","resource":"shared-work","provides":["dependency"],"requires":[],"adapter":"identity","config":{}})).unwrap());
    m.drivers[0].requires.push("dependency".into());
    m.grants = vec!["work".into(), "dependency".into()];
    m.validate().unwrap();
    assert_eq!(m.provider("dependency").unwrap().id, "two");
    m.grants.push("not-required".into());
    assert!(m.validate().is_err());
}
#[test]
fn unresolved_and_ambiguous_dependencies_rejected() {
    let mut m = manifest();
    m.requires.push("missing".into());
    assert!(m.validate().is_err());
    let mut m = manifest();
    m.drivers[0].requires.push("missing".into());
    assert!(m.validate().is_err());
    let mut m = manifest();
    m.drivers[0].provides.push("work".into());
    assert!(m.validate().is_err());
}
#[test]
fn dependency_cycles_are_rejected_before_dispatch() {
    let mut m = manifest();
    m.drivers[0].requires.push("work".into());
    assert!(m.validate().is_err());
    let mut m = manifest();
    m.drivers.push(serde_json::from_value(json!({"id":"two","resource":"shared-work","provides":["dependency"],"requires":["work"],"adapter":"identity","config":{}})).unwrap());
    m.drivers[0].requires.push("dependency".into());
    assert!(m.validate().is_err());
}
#[test]
fn builtin_read_capabilities_are_reserved_and_never_provider_dependencies() {
    let mut m = manifest();
    m.requires.push("knowledge.read".into());
    m.grants.push("knowledge.read".into());
    m.validate().unwrap();
    m.drivers[0].provides.push("knowledge.read".into());
    assert!(m.validate().is_err());
    let mut m = manifest();
    m.drivers[0].requires.push("artifact.read".into());
    assert!(m.validate().is_err());
}
#[test]
fn zero_limits_and_unknown_manifest_fields_rejected() {
    let mut m = manifest();
    m.limits.max_actions = 0;
    assert!(m.validate().is_err());
    let mut m = manifest();
    m.limits.lease_ttl_seconds = 0;
    assert!(m.validate().is_err());
    let mut v = serde_json::to_value(manifest()).unwrap();
    v["ambient_authority"] = json!(true);
    assert!(serde_json::from_value::<Manifest>(v).is_err());
}

#[test]
fn dependency_composition_cannot_cross_the_reserved_resource() {
    let mut m = manifest();
    m.drivers.push(serde_json::from_value(json!({"id":"two","resource":"another-resource","provides":["dependency"],"requires":[],"adapter":"identity","config":{}})).unwrap());
    m.drivers[0].requires.push("dependency".into());
    assert!(m.validate().is_err());
    m.drivers[1].resource = Some("shared-work".into());
    m.validate().unwrap();
}
