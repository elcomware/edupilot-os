import type { ButtonHTMLAttributes, HTMLAttributes } from "react";
export function Button({
  className = "",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button type="button" {...props} className={("btn " + className).trim()} />
  );
}
export function Card({
  className = "",
  ...props
}: HTMLAttributes<HTMLElement>) {
  return (
    <section {...props} className={("collection-sheet " + className).trim()} />
  );
}
export function Skeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-busy="true" className="account-skeleton">
      <span>{label}</span>
      <i aria-hidden="true" />
      <i aria-hidden="true" />
    </div>
  );
}
