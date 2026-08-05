import {forwardRef, type HTMLAttributes} from 'react'
import {cn} from '../../lib/utils'

export interface DividerProps extends HTMLAttributes<HTMLDivElement> {
  orientation?: 'horizontal' | 'vertical'
}

/** Snap Divider — a 2px outline-tinted rule; structural, no shadow. */
export const Divider = forwardRef<HTMLDivElement, DividerProps>(
  ({className, orientation = 'horizontal', ...props}, ref) => (
    <div
      ref={ref}
      role="separator"
      aria-orientation={orientation}
      className={cn(
        'border-snap-outline/25',
        orientation === 'horizontal' ? 'h-0 w-full border-t-2' : 'w-0 self-stretch border-l-2',
        className,
      )}
      {...props}
    />
  ),
)
Divider.displayName = 'Divider'
