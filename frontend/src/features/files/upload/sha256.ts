/**
 * Compute SHA-256 of a File using Web Crypto. For MVP we read the whole file
 * into memory; storage-service caps blobs at 5 GiB anyway. Larger files would
 * benefit from a streamed/incremental SHA-256 (browser doesn't expose one
 * natively, would require a wasm crate or a JS impl).
 */
export async function sha256Hex(file: File): Promise<string> {
  const buf = await file.arrayBuffer();
  const digest = await crypto.subtle.digest("SHA-256", buf);
  return bufToHex(digest);
}

function bufToHex(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let out = "";
  for (let i = 0; i < bytes.length; i++) {
    const b = bytes[i] as number;
    out += b.toString(16).padStart(2, "0");
  }
  return out;
}
