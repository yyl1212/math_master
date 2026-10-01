import { getGoClient } from "@/lib/api/server-client";
import { LearningHub } from "@/components/learning-hub";
export const dynamic = "force-dynamic";
export default async function Page() {
  const result = await getGoClient().listDomains({
    q: "",
    limit: 100,
    offset: 0,
  });
  return <LearningHub result={result} />;
}
