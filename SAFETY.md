# SAFETY

> **Work in progress.** This software is largely untested in the field. The
> core logic and unit tests pass, but end-to-end device testing is ongoing.
> Treat every operation as experimental until this notice is removed. Feedback
> and bug reports are welcome.

Cross-flashing can **brick hardware**. This tool is built to make that nearly
impossible, but firmware work is never zero-risk. Read this before you use it.

## The two invariants (why UART is usually unnecessary)
1. **Writes go to the INACTIVE A/B slot.** The working slot stays bootable, so a
   bad image is undone by a factory-reset-button hold — no UART.
2. **The bootloader env is APPEND-ONLY.** The tool only adds/updates individual
   fields on an env it has verified COMPLETE. It **cannot** erase the env or save
   a partial one. A *valid-but-incomplete* env is the single thing that turns a
   running AP into a bootloader brick — so the tool is structurally unable to make
   one. (This is the exact mistake the project was born from.)

## Always
- Let the tool take its **backup bundle** (mtd7/8/11 + config + hashes) first.
- Treat **ART** (calibration + factory MACs) as **read-only** — never write it.
- Keep pristine, unpatched stock images as your rollback path.
- For the first flash on a new model, keep a UART adapter attached.

## Scope & legal
Unofficial; **not affiliated with, endorsed by, or supported by EnGenius or
Senao.** Those names are used only to identify the affected products. For
**interoperability and self-hosting on hardware you own**. Do **not** use this for
warranty fraud, evading paid licensing on hardware you don't own, or defeating
theft protection. No vendor firmware is distributed here — you supply your own
images. Cross-flashing and synthetic serials are **unsupported/internal** and may
void warranty/support. No warranty; use at your own risk. Licensed under Blue Oak
Model License 1.0.0.
