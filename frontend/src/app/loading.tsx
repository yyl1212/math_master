export default function Loading() {
  return (
    <div className="content-state" role="status">
      <span className="loading-dot" aria-hidden="true" />
      <p>Loading content…</p>
    </div>
  );
}
