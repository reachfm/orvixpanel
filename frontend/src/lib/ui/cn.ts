import { clsx } from "clsx";

/**
 * Small `cn` helper — wraps clsx so call sites stay terse.
 * Equivalent to the popular shadcn `cn` utility.
 */
export function cn(...args: Parameters<typeof clsx>): string {
  return clsx(...args);
}
