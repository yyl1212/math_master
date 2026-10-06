"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
const messages = {
  empty: uiMessage("public.state.empty",{}),
  "no-results": uiMessage("public.state.noResults",{}),
  "not-found": uiMessage("public.state.notFound",{}),
  unavailable: uiMessage("public.state.unavailable",{}),
};
export function ContentState({ kind }: { kind: keyof typeof messages }) {
  return (
    <section
      className="content-state"
      role={kind === "unavailable" ? "alert" : "status"}
    >
      <span className="state-symbol" aria-hidden="true">
        {kind === "unavailable" ? "↻" : "◇"}
      </span>
      <h2><UiText notice={messages[kind]}/></h2>
      <p>
        <UiText notice={uiMessage(({empty:"public.help.empty","no-results":"public.help.noResults","not-found":"public.help.notFound",unavailable:"public.help.unavailable"} as const)[kind],{})}/>
      </p>
      {kind === "unavailable" ? (
        <button
          className="button secondary"
          onClick={() => window.location.reload()}
        ><UiText notice={uiMessage("content-state.try.again.d8b839",{})}/></button>
      ) : (
        <a className="button secondary" href="/knowledge"><UiText notice={uiMessage("content-state.explore.knowledge.7bd0d2",{})}/></a>
      )}
    </section>
  );
}
