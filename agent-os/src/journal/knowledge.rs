use super::Journal;
use anyhow::{ensure, Context, Result};
use rusqlite::params;
use serde_json::{json, Value};

impl Journal {
    pub fn add_knowledge(&self, node: &Value) -> Result<()> {
        let id = node["id"]
            .as_str()
            .filter(|v| !v.is_empty())
            .context("knowledge id must be nonempty")?;
        let kind = node["kind"].as_str().context("knowledge kind required")?;
        ensure!(
            ["decision", "finding", "hypothesis", "disproved"].contains(&kind),
            "invalid knowledge kind"
        );
        let status = node["status"]
            .as_str()
            .context("knowledge status required")?;
        ensure!(
            ["draft", "stable", "deprecated"].contains(&status),
            "invalid knowledge status"
        );
        let trust = node["trust"].as_str().context("knowledge trust required")?;
        ensure!(
            ["unverified", "machine", "human"].contains(&trust),
            "invalid knowledge trust"
        );
        if status == "draft" || !node["stale_after"].is_null() {
            ensure!(
                node["stale_after"].as_i64().is_some_and(|time| time > 0),
                "drafts require a positive expiry timestamp"
            );
        }
        self.validate_sources(node)?;
        self.connection.execute(
            "INSERT INTO knowledge(id,body) VALUES (?1,?2) ON CONFLICT(id) DO UPDATE SET body=excluded.body",
            params![id, node.to_string()],
        )?;
        Ok(())
    }

    fn validate_sources(&self, node: &Value) -> Result<()> {
        for source in node["sources"]
            .as_array()
            .context("knowledge sources must be an array")?
        {
            let hash = source
                .as_str()
                .or_else(|| source["sha256"].as_str())
                .context("source must reference an artifact hash")?;
            self.get_artifact(hash)?;
        }
        Ok(())
    }

    pub fn context(&self, now: i64) -> Result<Value> {
        let mut confirmed = Vec::new();
        let mut provisional = Vec::new();
        let mut statement = self
            .connection
            .prepare("SELECT body FROM knowledge ORDER BY id")?;
        let rows = statement.query_map([], |row| row.get::<_, String>(0))?;
        for row in rows {
            let node: Value = serde_json::from_str(&row?)?;
            if node["status"] == "deprecated"
                || node["stale_after"]
                    .as_i64()
                    .is_some_and(|expiry| expiry <= now)
            {
                continue;
            }
            self.validate_sources(&node)?;
            if node["status"] == "stable" && node["trust"] == "human" {
                confirmed.push(node);
            } else {
                provisional.push(node);
            }
        }
        Ok(json!({"confirmed":confirmed,"provisional":provisional}))
    }
}

impl crate::storage::KnowledgeStore for Journal {
    fn context(&self, now: i64) -> Result<Value> {
        Journal::context(self, now)
    }
}
