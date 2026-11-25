use super::{EvidenceStore, Journal};
use crate::storage::ArtifactStore;
use anyhow::{ensure, Context, Result};
use sha2::{Digest, Sha256};
use std::{
    fs::{self, OpenOptions},
    io::Write,
    path::{Path, PathBuf},
    sync::atomic::{AtomicU64, Ordering},
};

/// Filesystem CAS adapter. The path is owned here, never by the SQL journal.
pub struct LocalCas {
    cas: PathBuf,
}

impl LocalCas {
    pub fn open(dir: &Path) -> Result<Self> {
        fs::create_dir_all(dir)?;
        Ok(Self {
            cas: dir.to_path_buf(),
        })
    }

    pub fn put_artifact(&self, bytes: &[u8]) -> Result<String> {
        let hash = format!("{:x}", Sha256::digest(bytes));
        let destination = self.cas.join(&hash);
        if destination.exists() {
            self.get_artifact(&hash)?;
            return Ok(hash);
        }
        static SEQUENCE: AtomicU64 = AtomicU64::new(0);
        let temporary = self.cas.join(format!(
            ".put-{}-{}",
            std::process::id(),
            SEQUENCE.fetch_add(1, Ordering::Relaxed)
        ));
        let result = (|| -> Result<()> {
            let mut file = OpenOptions::new()
                .create_new(true)
                .write(true)
                .open(&temporary)?;
            file.write_all(bytes)?;
            file.sync_all()?;
            match fs::hard_link(&temporary, &destination) {
                Ok(()) => (),
                Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                    self.get_artifact(&hash)?;
                }
                Err(error) => return Err(error.into()),
            }
            fs::File::open(&self.cas)?.sync_all()?;
            Ok(())
        })();
        let _ = fs::remove_file(&temporary);
        result?;
        Ok(hash)
    }

    pub fn get_artifact(&self, hash: &str) -> Result<Vec<u8>> {
        ensure!(
            hash.len() == 64
                && hash
                    .bytes()
                    .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b)),
            "invalid artifact hash"
        );
        let bytes = fs::read(self.cas.join(hash)).context("artifact is unavailable")?;
        ensure!(
            format!("{:x}", Sha256::digest(&bytes)) == hash,
            "artifact checksum mismatch"
        );
        Ok(bytes)
    }
}

impl Journal {
    pub fn put_artifact(&self, bytes: &[u8]) -> Result<String> {
        let hash = self.artifacts.put_artifact(bytes)?;
        ensure!(
            format!("{:x}", Sha256::digest(bytes)) == hash,
            "evidence store returned an incorrect content address"
        );
        ensure!(
            self.get_artifact(&hash)? == bytes,
            "evidence store did not preserve written bytes"
        );
        Ok(hash)
    }
    pub fn get_artifact(&self, hash: &str) -> Result<Vec<u8>> {
        ensure!(
            hash.len() == 64
                && hash
                    .bytes()
                    .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b)),
            "invalid artifact hash"
        );
        let bytes = self.artifacts.get_artifact(hash)?;
        ensure!(
            format!("{:x}", Sha256::digest(&bytes)) == hash,
            "artifact checksum mismatch"
        );
        Ok(bytes)
    }
}
impl ArtifactStore for LocalCas {
    fn get_artifact(&self, hash: &str) -> Result<Vec<u8>> {
        LocalCas::get_artifact(self, hash)
    }
}
impl EvidenceStore for LocalCas {
    fn put_artifact(&self, bytes: &[u8]) -> Result<String> {
        LocalCas::put_artifact(self, bytes)
    }
}
impl ArtifactStore for Journal {
    fn get_artifact(&self, hash: &str) -> Result<Vec<u8>> {
        Journal::get_artifact(self, hash)
    }
}
