"use client";
const messages = {
  empty: "The catalogue is being prepared.",
  "no-results": "No domains match your search.",
  "not-found": "This content is not available.",
  unavailable: "Content is temporarily unavailable.",
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
      <h2>{messages[kind]}</h2>
      <p>
        {kind === "empty"
          ? "Learning domains will appear here as the catalogue takes shape."
          : kind === "no-results"
            ? "Try another topic, or explore all learning domains."
            : kind === "not-found"
              ? "Explore the knowledge map to find available content."
              : "Please try again in a moment."}
      </p>
      {kind === "unavailable" ? (
        <button
          className="button secondary"
          onClick={() => window.location.reload()}
        >
          Try again
        </button>
      ) : (
        <a className="button secondary" href="/knowledge">
          Explore knowledge
        </a>
      )}
    </section>
  );
}
