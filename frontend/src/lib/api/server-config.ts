import "server-only";
export function parseGoOrigin(raw: string): string {
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    throw new Error("Invalid Go API configuration.");
  }
  if (
    !["http:", "https:"].includes(url.protocol) ||
    url.username ||
    url.password ||
    url.pathname !== "/" ||
    url.search ||
    url.hash
  )
    throw new Error("Invalid Go API configuration.");
  return url.origin;
}
export function getGoOrigin(): string {
  return parseGoOrigin(process.env.GO_API_INTERNAL_URL ?? "");
}
