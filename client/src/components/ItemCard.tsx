import type { VaultItem } from "../lib/types";

interface Props {
  item: VaultItem;
  onClick: () => void;
}

export default function ItemCard({ item, onClick }: Props) {
  const { name, username } = item.data;
  const title = name || "(no name)";

  return (
    <button
      type="button"
      onClick={onClick}
      className="flex w-full items-center gap-3 rounded-md border border-gray-200 px-3 py-2.5 text-left transition-colors hover:bg-gray-50"
    >
      <span
        aria-hidden
        className="grid h-9 w-9 shrink-0 place-items-center rounded-md bg-gray-100 text-sm font-medium uppercase text-gray-600"
      >
        {title[0]}
      </span>
      <span className="min-w-0">
        <span className="block truncate text-sm font-medium">{title}</span>
        {username && (
          <span className="block truncate text-sm text-gray-500">
            {username}
          </span>
        )}
      </span>
    </button>
  );
}
