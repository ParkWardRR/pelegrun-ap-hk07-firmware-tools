//! `quarry` — CLI for the Senao/EnGenius header re-head + Code27 serial core.
//!
//! Unofficial. Operates only on image files you supply; touches no device.

use std::process::ExitCode;

fn usage() -> &'static str {
    "quarry — Senao/EnGenius ap-hk07 firmware header + serial tool (unofficial)

USAGE:
  quarry inspect <image.bin>
  quarry rehead  <in.bin> <out.bin> --to <product_id>     # e.g. --to 282
  quarry serial  --model <CODE> [--prefix PPPP] [--suffix SSSS]
  quarry snextra --model <CODE> [--prefix PPPP]           # 20-char field 19
  quarry check   <serial12>

Product ids: 282 = EWS377AP v3, 300 = EWS377-FIT, 284 = ECW230v3.
Model codes:  X44 = EWS377AP v3, X45 = EWS377-FIT, X42 = ECW230v3."
}

fn arg_val(args: &[String], key: &str) -> Option<String> {
    args.iter()
        .position(|a| a == key)
        .and_then(|i| args.get(i + 1).cloned())
}

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let cmd = args.first().map(String::as_str).unwrap_or("");
    let rest = if args.is_empty() { &[][..] } else { &args[1..] };

    let result: Result<(), String> = match cmd {
        "inspect" => cmd_inspect(rest),
        "rehead" => cmd_rehead(rest),
        "serial" => cmd_serial(rest),
        "snextra" => cmd_snextra(rest),
        "check" => cmd_check(rest),
        "-h" | "--help" | "help" | "" => {
            println!("{}", usage());
            Ok(())
        }
        other => Err(format!("unknown command: {other}\n\n{}", usage())),
    };

    match result {
        Ok(()) => ExitCode::SUCCESS,
        Err(e) => {
            eprintln!("error: {e}");
            ExitCode::FAILURE
        }
    }
}

fn cmd_inspect(a: &[String]) -> Result<(), String> {
    let path = a.first().ok_or("inspect: missing <image.bin>")?;
    let data = std::fs::read(path).map_err(|e| format!("read {path}: {e}"))?;
    let h = quarry::header::parse(&data).map_err(|e| e.to_string())?;
    println!("file        : {path} ({} bytes)", data.len());
    println!("vendor_id   : {}", h.vendor_id);
    println!("product_id  : {} ({})", h.product_id, h.product().label());
    println!("fw_type     : {}", h.firmware_type);
    println!("model       : {}", h.model);
    println!("magic       : {:#010x} (ok)", h.magic);
    Ok(())
}

fn cmd_rehead(a: &[String]) -> Result<(), String> {
    let input = a.first().ok_or("rehead: missing <in.bin>")?;
    let output = a.get(1).ok_or("rehead: missing <out.bin>")?;
    let to: u32 = arg_val(a, "--to")
        .ok_or("rehead: missing --to <product_id>")?
        .parse()
        .map_err(|_| "rehead: --to must be a number")?;
    let mut data = std::fs::read(input).map_err(|e| format!("read {input}: {e}"))?;
    let old = quarry::header::rehead(&mut data, to).map_err(|e| e.to_string())?;
    std::fs::write(output, &data).map_err(|e| format!("write {output}: {e}"))?;
    println!(
        "re-headed product_id {} -> {} : {} bytes -> {}",
        old,
        to,
        data.len(),
        output
    );
    println!("note: verify on a recoverable A/B slot; the tool never asserts a flash succeeded.");
    Ok(())
}

fn cmd_serial(a: &[String]) -> Result<(), String> {
    let model = arg_val(a, "--model").ok_or("serial: missing --model <CODE>")?;
    let prefix = arg_val(a, "--prefix").unwrap_or_else(|| "SWLW".to_string());
    let suffix = arg_val(a, "--suffix").unwrap_or_else(|| "0001".to_string());
    let s = quarry::serial::make_serial(&prefix, &model, &suffix).map_err(|e| e.to_string())?;
    println!("{s}");
    Ok(())
}

fn cmd_snextra(a: &[String]) -> Result<(), String> {
    let model = arg_val(a, "--model").ok_or("snextra: missing --model <CODE>")?;
    let prefix = arg_val(a, "--prefix").unwrap_or_default();
    let s = quarry::serial::make_snextra(&prefix, &model).map_err(|e| e.to_string())?;
    println!("{s}");
    Ok(())
}

fn cmd_check(a: &[String]) -> Result<(), String> {
    let s = a.first().ok_or("check: missing <serial12>")?;
    let ok = quarry::serial::validate_serial(s);
    let mc = quarry::serial::model_code(s)
        .map(str::to_string)
        .unwrap_or_else(|_| "?".into());
    println!("serial    : {s}");
    println!("valid     : {ok}");
    println!("model_code: {mc}");
    if ok {
        Ok(())
    } else {
        Err("check character does not match".into())
    }
}
