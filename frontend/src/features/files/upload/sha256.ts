import { sha256 } from "js-sha256";

/**
 * Compute SHA-256 of a File. Uses js-sha256 library which works in non-secure
 * contexts (http://), unlike crypto.subtle which requires https or localhost.
 * For MVP we read the whole file into memory; storage-service caps blobs at
 * 5 GiB anyway. Larger files would benefit from a streamed/incremental approach.
 */
export async function sha256Hex(file: File): Promise<string> {
  const buf = await file.arrayBuffer();
  return sha256(buf);
}
