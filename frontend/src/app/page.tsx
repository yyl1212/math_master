import { getGoClient } from "@/lib/api/server-client";
export const dynamic = "force-dynamic";
export default async function Page() {
  const r = await getGoClient().listDomains({ q: "", limit: 100, offset: 0 });
  return (
    <main>
      {r.ok
        ? `${r.data.total} learning domains`
        : "Content is temporarily unavailable."}
    </main>
  );
}
