use base64::{engine::general_purpose, Engine};
use hmac::{Hmac, Mac};
use sha2::Sha256;

type HmacSha256 = Hmac<Sha256>;

// 値にMACを付与して改竄検知する簡易方式 (暗号化しない)。
pub fn seal_token(key_bytes: &[u8; 32], token: &str) -> String {
    let mut mac = HmacSha256::new_from_slice(key_bytes).expect("HMAC key");
    mac.update(token.as_bytes());
    let tag = mac.finalize().into_bytes();
    let mut data = token.as_bytes().to_vec();
    data.extend_from_slice(&tag);
    general_purpose::STANDARD_NO_PAD.encode(data)
}

pub fn open_token(key_bytes: &[u8; 32], sealed: &str) -> Option<String> {
    let data = general_purpose::STANDARD_NO_PAD.decode(sealed).ok()?;
    if data.len() < 32 { return None; }
    let (msg, tag) = data.split_at(data.len() - 32);
    let mut mac = HmacSha256::new_from_slice(key_bytes).ok()?;
    mac.update(msg);
    mac.verify_slice(tag).ok()?;
    String::from_utf8(msg.to_vec()).ok()
}

pub fn derive_key_from_env(secret: &str) -> [u8; 32] {
    use sha2::Digest;
    let mut hasher = Sha256::new();
    hasher.update(secret.as_bytes());
    let out = hasher.finalize();
    let mut key = [0u8; 32];
    key.copy_from_slice(&out);
    key
}
