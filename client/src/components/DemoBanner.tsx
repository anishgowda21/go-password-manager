export default function DemoBanner() {
  return (
    <div
      role="note"
      className="mb-6 rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900"
    >
      <p className="font-medium">⚠️ Experimental demo</p>
      <p className="mt-1 text-amber-800">
        This is a learning project, not a production app. It is designed around
        zero-knowledge, end-to-end encrypted storage, but it has not been
        audited and your data may be wiped at any time. Do not store real
        passwords or sensitive information.
      </p>
    </div>
  );
}
