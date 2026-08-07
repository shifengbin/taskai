import {forwardRef, type ButtonHTMLAttributes} from 'react'
import {Slot} from '@radix-ui/react-slot'
import {cva} from 'class-variance-authority'
import {cn, focusRing} from '../../lib/utils'

const iconButtonSizes = cva('', {
  variants: {
    size: {
      md: 'h-8 w-8',
      sm: 'h-7 w-7',
    },
  },
  defaultVariants: {size: 'md'},
})

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  asChild?: boolean
  size?: 'sm' | 'md'
  /** Accessible label — icon buttons must not be empty for screen readers. */
  'aria-label': string
}

/**
 * Nebula IconButton — 32px square (28px at size="sm"), fine outline and soft glow.
 */
export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  ({className, asChild = false, size = 'md', type, ...props}, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        ref={ref}
        type={asChild ? undefined : (type ?? 'button')}
        className={cn(
          'inline-grid place-items-center shrink-0',
          iconButtonSizes({size}),
          'border border-snap-outline rounded-snap bg-snap-surface text-snap-ink',
          'shadow-snap-sm transition-[transform,box-shadow] duration-150',
          'hover:-translate-x-px hover:-translate-y-px hover:shadow-snap hover:text-snap-cobalt',
          focusRing,
          'disabled:pointer-events-none disabled:opacity-50',
          'motion-reduce:transition-none motion-reduce:translate-x-0 motion-reduce:translate-y-0',
          className,
        )}
        {...props}
      />
    )
  },
)
IconButton.displayName = 'IconButton'
