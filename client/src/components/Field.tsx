import type { ReactNode } from "react";

interface Props {
  label: string;
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  multiline?: boolean;
  type?: string;
  autoComplete?: string;
  action?: ReactNode;
}

const base =
  "w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none transition-colors focus:border-gray-900 read-only:bg-gray-50 read-only:text-gray-700";

export default function Field({
  label,
  value,
  onChange,
  readOnly,
  multiline,
  type = "text",
  autoComplete,
  action,
}: Props) {
  const control = multiline ? (
    <textarea
      className={`${base} min-h-20 max-h-60 resize-y`}
      rows={3}
      value={value}
      readOnly={readOnly}
      onChange={(e) => onChange(e.target.value)}
    />
  ) : (
    <input
      className={base}
      type={type}
      value={value}
      readOnly={readOnly}
      autoComplete={autoComplete}
      onChange={(e) => onChange(e.target.value)}
    />
  );

  return (
    <label className="block text-sm text-gray-600">
      <span className="mb-1 block">{label}</span>
      {action ? (
        <div className="flex gap-2">
          {control}
          {action}
        </div>
      ) : (
        control
      )}
    </label>
  );
}
