import {clsx, type ClassValue} from 'clsx'
import {twMerge} from 'tailwind-merge'

/** Merge Tailwind classes with conflict resolution (shadcn pattern). */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}

/**
 * Cobalt 3px focus ring, the single Snap focus affordance.
 * Applied via focus-visible so mouse clicks don't show it.
 */
export const focusRing =
  'outline-none focus-visible:ring-[3px] focus-visible:ring-snap-cobalt'
