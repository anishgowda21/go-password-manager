import type { ButtonHTMLAttributes } from "react";

const variants = {
  default:
    "rounded-md border border-gray-300 px-3 py-2 text-sm font-medium hover:bg-gray-100",
  primary:
    "rounded-md border border-gray-900 bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-black",
  danger: "text-sm text-red-600 hover:underline",
};

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: keyof typeof variants;
}

export default function Button({
  variant = "default",
  type = "button",
  className = "",
  ...props
}: Props) {
  return (
    <button
      type={type}
      className={`${variants[variant]} transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${className}`}
      {...props}
    />
  );
}
