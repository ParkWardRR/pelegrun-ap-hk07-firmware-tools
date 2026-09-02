//! shots — spawn the built swallow/quarry binaries in a PTY (via termwright),
//! assert on-screen text (terminal E2E), and export PNG screenshots for the README.
//!
//! Env: SWALLOW, QUARRY = paths to the built binaries. OUT = screenshot dir.
use anyhow::{Context, Result};
use std::time::Duration;
use termwright::prelude::*;

async fn shot(name: &str, prog: &str, args: &[&str], expect: &str, rows: u16) -> Result<()> {
    let term = Terminal::builder()
        .size(92, rows)
        .spawn(prog, args)
        .await
        .with_context(|| format!("spawn {prog}"))?;
    // wait for the marker, then let the screen settle
    let _ = term.expect(expect).timeout(Duration::from_secs(8)).await;
    tokio::time::sleep(Duration::from_millis(300)).await;
    let screen = term.screen().await;
    anyhow::ensure!(
        screen.contains(expect),
        "E2E FAIL: {name}: screen missing {expect:?}"
    );
    let out = std::env::var("OUT").unwrap_or_else(|_| "docs/screenshots".into());
    let path = format!("{out}/{name}.png");
    term.screenshot().await.save(&path)?;
    println!("  ✔ {name}: asserted {expect:?} · wrote {path}");
    Ok(())
}

#[tokio::main]
async fn main() -> Result<()> {
    let swallow = std::env::var("SWALLOW").context("set SWALLOW=path/to/swallow")?;
    let quarry = std::env::var("QUARRY").context("set QUARRY=path/to/quarry")?;
    println!("shots — terminal E2E + screenshots (termwright)");

    shot("swallow-demo", &swallow, &["demo"], "invariants held", 40).await?;
    shot("swallow-serial", &swallow, &["serial", "--model", "X42", "--prefix", "EPC1"], "EPC1X4200011", 8).await?;
    shot("swallow-plan", &swallow, &["plan"], "APPEND-ONLY", 14).await?;
    shot("quarry-inspect", &quarry, &["inspect", "/tmp/sample-hk07.bin"], "EWS377AP v3", 12).await?;

    println!("all screens asserted + captured.");
    Ok(())
}
