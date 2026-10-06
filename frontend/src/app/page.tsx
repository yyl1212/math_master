import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.home");}
import { getGoClient } from "@/lib/api/server-client";
import { LearningHub } from "@/components/learning-hub";
export const dynamic = "force-dynamic";
export default async function Page() {
  const result = await getGoClient().listDomains({
    q: "",
    limit: 100,
    offset: 0,
  });
  return <><UiPageTitle messageKey="page.home"/>{<LearningHub result={result} />}</>;
}
