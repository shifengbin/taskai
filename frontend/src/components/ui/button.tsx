import {forwardRef, type ButtonHTMLAttributes} from 'react'
import {Slot} from '@radix-ui/react-slot'
import {cva, type VariantProps} from 'class-variance-authority'
import {cn, focusRing} from '../../lib/utils'

/**
 * Snap Button — 2px outline, hard offset shadow, hover lift.
 * `coral`/`danger` fills only ever carry bold white text (contrast ≥3:1 large-bold).
 * `primary` (cobalt) passes 4.5:1 for normal text.
 */
const buttonVariants = cva(
  [
    'inline-flex items-center justify-center gap-2 whitespace-nowrap select-none',
    'font-sans font-bold leading-none border-2 border-snap-outline rounded-snap',
    'shadow-snap-sm transition-[transform,box-shadow] duration-150',
    'hover:-translate-x-px hover:-translate-y-px',
    focusRing,
    'disabled:pointer-events-none disabled:opacity-50 motion-reduce:transition-none motion-reduce:translate-x-0 motion-reduce:translate-y-0',
  ],
  {
    variants: {
      variant: {
        primary: 'bg-snap-cobalt text-white hover:shadow-snap',
        secondary: 'bg-snap-surface text-snap-ink hover:shadow-snap',
        danger: 'bg-snap-error text-white hover:shadow-snap',
        coral: 'bg-snap-coral text-white hover:shadow-snap',
        amber: 'bg-snap-amber text-snap-ink hover:shadow-snap',
        ghost: 'bg-transparent border-transparent shadow-none text-snap-ink hover:bg-snap-surface-2',
      },
      size: {
        sm: 'h-8 px-3 text-xs',
        md: 'h-10 px-4 text-sm',
        lg: 'h-12 px-6 text-base',
        icon: 'h-8 w-8 p-0',
      },
    },
    defaultVariants: {variant: 'secondary', size: 'md'},
  },
)

export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({className, variant, size, asChild = false, type, ...props}, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        ref={ref}
        type={asChild ? undefined : (type ?? 'button')}
        className={cn(buttonVariants({variant, size}), className)}
        {...props}
      />
    )
  },
)
Button.displayName = 'Button'

export {buttonVariants}
