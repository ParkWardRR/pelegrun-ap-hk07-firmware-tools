//! Drive the swallow TUI with termwright: assert each screen and snapshot PNGs.
//!
//! Env: SWALLOW_BIN (default /tmp/swallow), SHOT_DIR (default docs/screenshots).
use std::time::Duration;
use termwright::prelude::*;

const DOWN: &[u8] = b"\x1b[B"; // down-arrow escape

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let bin = std::env::var("SWALLOW_BIN").unwrap_or_else(|_| "/tmp/swallow".into());
    let dir = std::env::var("SHOT_DIR").unwrap_or_else(|_| "docs/screenshots".into());
    std::fs::create_dir_all(&dir)?;

    // (file stem, a substring that must be on-screen for that menu item)
    let screens = [
        ("01-discover", "Find the AP"),
        ("02-connect", "Pick the access path"),
        ("03-backup", "hard gate"),
        ("04-safeguards", "append-only"),
        ("05-identity", "unique, valid serial"),
        ("06-install", "No UART, no open case"),
        ("07-verify", "exactly as intended"),
    ];

    let term = Terminal::builder().size(100, 30).spawn(&bin, &[]).await?;
    term.expect("swallow").timeout(Duration::from_secs(8)).await?;

    let mut failures = 0;
    for (i, (stem, marker)) in screens.iter().enumerate() {
        term.wait_idle(Duration::from_millis(250)).timeout(Duration::from_secs(3)).await.ok();
        let screen = term.screen().await;
        if screen.contains(marker) {
            println!("PASS  {:<14} contains {:?}", stem, marker);
        } else {
            eprintln!("FAIL  {:<14} MISSING {:?}", stem, marker);
            failures += 1;
        }
        let path = format!("{dir}/{stem}.png");
        term.screenshot().await.save(&path)?;
        println!("shot  {path}");
        if i + 1 < screens.len() {
            term.send_raw(DOWN).await?; // next menu item
        }
    }

    term.send_raw(b"q").await.ok();
    term.kill().await.ok();

    if failures > 0 {
        anyhow::bail!("{failures} screen assertion(s) failed");
    }
    println!("\nOK — {} screens asserted + snapshotted", screens.len());
    Ok(())
}
