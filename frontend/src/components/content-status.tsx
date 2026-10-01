export function ContentStatus({ status }: { status: "planned" | "published" }) {
  return (
    <span className={`status status-${status}`}>
      <span aria-hidden="true" />
      {status === "published" ? "Published" : "In development"}
    </span>
  );
}
