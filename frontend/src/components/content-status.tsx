"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage} from "@/lib/i18n/format";
export function ContentStatus({ status }: { status: "planned" | "published" }) {
  return (
    <span className={`status status-${status}`}>
      <span aria-hidden="true" />
      <UiText notice={uiMessage(status==="published"?"knowledge-map.published.2ef42e":"knowledge-map.in.development.5259ae",{})}/>
    </span>
  );
}
