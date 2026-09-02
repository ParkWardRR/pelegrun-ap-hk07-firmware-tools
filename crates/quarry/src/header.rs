//! Senao image header: parse, validate, and one-field `product_id` re-head.
//!
//! All multi-byte fields are **big-endian**. Offsets are absolute file offsets,
//! verified on real EWS377AP v3 / EWS377-FIT / ECW230v3 images.

use crate::Error;

/// `vendor_id` (u32 BE). `257` across the ap-hk07 line.
pub const OFF_VENDOR_ID: usize = 0x04;
/// `product_id` (u32 BE) — **the one field the re-head patches**.
pub const OFF_PRODUCT_ID: usize = 0x08;
/// `firmware_type` (u32 BE) — `0` = combo image (has CAPWAP sub-header).
pub const OFF_FIRMWARE_TYPE: usize = 0x1C;
/// `md5sum` (16 bytes) — over the payload, unaffected by a header re-head.
pub const OFF_MD5SUM: usize = 0x28;
/// `chksum` (u32 BE) — vendor-proprietary; not read by the upgrade path.
pub const OFF_CHKSUM: usize = 0x58;
/// `magic` (u32 BE) — must be `0x12345678`.
pub const OFF_MAGIC: usize = 0x5C;
/// `model` string (ASCII, NUL/garbage-terminated).
pub const OFF_MODEL: usize = 0x88;

/// The Senao magic value.
pub const MAGIC: u32 = 0x1234_5678;

/// Smallest image length we can fully parse (covers the model string field).
pub const MIN_LEN: usize = OFF_MODEL + 16;

/// Known `product_id` values on the ap-hk07 board family.
///
/// The three cross-flash siblings are `282`/`300`/`284`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Product {
    Ews377ApV3, // 282
    Ews377Fit,  // 300
    Ecw230V3,   // 284
    Other(u32),
}

impl Product {
    pub fn from_id(id: u32) -> Self {
        match id {
            282 => Product::Ews377ApV3,
            300 => Product::Ews377Fit,
            284 => Product::Ecw230V3,
            other => Product::Other(other),
        }
    }
    pub fn id(self) -> u32 {
        match self {
            Product::Ews377ApV3 => 282,
            Product::Ews377Fit => 300,
            Product::Ecw230V3 => 284,
            Product::Other(v) => v,
        }
    }
    pub fn label(self) -> &'static str {
        match self {
            Product::Ews377ApV3 => "EWS377AP v3",
            Product::Ews377Fit => "EWS377-FIT",
            Product::Ecw230V3 => "ECW230v3",
            Product::Other(_) => "unknown",
        }
    }
}

/// A parsed, validated Senao header.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Header {
    pub vendor_id: u32,
    pub product_id: u32,
    pub firmware_type: u32,
    pub chksum: u32,
    pub magic: u32,
    pub model: String,
}

impl Header {
    pub fn product(&self) -> Product {
        Product::from_id(self.product_id)
    }
}

fn be_u32(data: &[u8], off: usize) -> Result<u32, Error> {
    let end = off + 4;
    if data.len() < end {
        return Err(Error::TooShort {
            need: end,
            got: data.len(),
        });
    }
    Ok(u32::from_be_bytes([
        data[off],
        data[off + 1],
        data[off + 2],
        data[off + 3],
    ]))
}

/// Parse and validate a Senao image header. Errors if the image is too short or
/// the magic is wrong (i.e. it isn't a Senao image).
pub fn parse(data: &[u8]) -> Result<Header, Error> {
    if data.len() < MIN_LEN {
        return Err(Error::TooShort {
            need: MIN_LEN,
            got: data.len(),
        });
    }
    let magic = be_u32(data, OFF_MAGIC)?;
    if magic != MAGIC {
        return Err(Error::BadMagic { got: magic });
    }
    let model_bytes = &data[OFF_MODEL..OFF_MODEL + 16];
    let end = model_bytes
        .iter()
        .position(|&b| b == 0 || !(0x20..=0x7e).contains(&b))
        .unwrap_or(model_bytes.len());
    let model = String::from_utf8_lossy(&model_bytes[..end]).into_owned();

    Ok(Header {
        vendor_id: be_u32(data, OFF_VENDOR_ID)?,
        product_id: be_u32(data, OFF_PRODUCT_ID)?,
        firmware_type: be_u32(data, OFF_FIRMWARE_TYPE)?,
        chksum: be_u32(data, OFF_CHKSUM)?,
        magic,
        model,
    })
}

/// The **one-field re-head**: overwrite `product_id` (4 bytes at 0x08) in place so
/// the image passes a sibling model's upload gate. Validates the header first,
/// returns the previous `product_id`.
///
/// The payload `md5sum` covers the untouched payload (stays valid) and `chksum`
/// is never read, so nothing else needs fixing.
pub fn rehead(data: &mut [u8], new_product_id: u32) -> Result<u32, Error> {
    let old = parse(data)?.product_id;
    data[OFF_PRODUCT_ID..OFF_PRODUCT_ID + 4].copy_from_slice(&new_product_id.to_be_bytes());
    Ok(old)
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Build a minimal synthetic Senao header for tests.
    fn synth(vendor: u32, product: u32, model: &str) -> Vec<u8> {
        let mut d = vec![0u8; 0x100];
        d[OFF_VENDOR_ID..OFF_VENDOR_ID + 4].copy_from_slice(&vendor.to_be_bytes());
        d[OFF_PRODUCT_ID..OFF_PRODUCT_ID + 4].copy_from_slice(&product.to_be_bytes());
        d[OFF_MAGIC..OFF_MAGIC + 4].copy_from_slice(&MAGIC.to_be_bytes());
        let mb = model.as_bytes();
        d[OFF_MODEL..OFF_MODEL + mb.len()].copy_from_slice(mb);
        d
    }

    #[test]
    fn parse_ok() {
        let d = synth(257, 284, "ECW230v3");
        let h = parse(&d).unwrap();
        assert_eq!(h.vendor_id, 257);
        assert_eq!(h.product_id, 284);
        assert_eq!(h.model, "ECW230v3");
        assert_eq!(h.product(), Product::Ecw230V3);
    }

    #[test]
    fn bad_magic_rejected() {
        let mut d = synth(257, 284, "ECW230v3");
        d[OFF_MAGIC] = 0xFF;
        assert!(matches!(parse(&d), Err(Error::BadMagic { .. })));
    }

    #[test]
    fn too_short_rejected() {
        assert!(matches!(parse(&[0u8; 8]), Err(Error::TooShort { .. })));
    }

    #[test]
    fn rehead_284_to_282() {
        // ECW230v3 (284) -> pass the EWS377AP v3 (282) gate.
        let mut d = synth(257, 284, "ECW230v3");
        let old = rehead(&mut d, 282).unwrap();
        assert_eq!(old, 284);
        assert_eq!(parse(&d).unwrap().product_id, 282);
        assert_eq!(parse(&d).unwrap().product(), Product::Ews377ApV3);
    }

    #[test]
    fn rehead_leaves_other_fields_intact() {
        let mut d = synth(257, 300, "EWS377-FIT");
        let before_magic = be_u32(&d, OFF_MAGIC).unwrap();
        rehead(&mut d, 282).unwrap();
        assert_eq!(be_u32(&d, OFF_MAGIC).unwrap(), before_magic);
        assert_eq!(be_u32(&d, OFF_VENDOR_ID).unwrap(), 257);
    }

    #[test]
    fn rehead_rejects_non_senao() {
        let mut junk = vec![0u8; 0x100];
        assert!(rehead(&mut junk, 282).is_err());
    }
}
