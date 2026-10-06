import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
export default function Loading() {
  return (
    <div className="content-state" role="status">
      <span className="loading-dot" aria-hidden="true" />
      <p><UiText notice={uiMessage("loading.loading.content.e2e410",{})}/></p>
    </div>
  );
}
