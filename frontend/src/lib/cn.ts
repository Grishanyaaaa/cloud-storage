import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/** Merge tailwind classes safely (last-wins). */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
