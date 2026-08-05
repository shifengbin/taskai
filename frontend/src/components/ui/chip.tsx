import {forwardRef, type HTMLAttributes} from 'react'
import {cva, type VariantProps} from 'class-variance-authority'
import {cn} from '../../lib/utils'

const chipVariants = cva(
  'inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-bold leading-5 border-2 border-snap-outline rounded-snap-sm whitespace-nowrap',
  {
    variants: {
      variant: {
        default: 'bg-snap-surface text-snap-ink',
        muted: 'bg-snap-surface-2 text-snap-muted',
        accent: 'bg-snap-coral text-white',
        info: 'bg-snap-cobalt text-white',
        amber: 'bg-snap-amber text-snap-ink',
      },
    },
    defaultVariants: {variant: 'default'},
  },
)

export interface ChipProps
  extends HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof chipVariants> {}

/** Snap Chip — small status/label pill, 2px outline, 6px radius. */
export const Chip = forwardRef<HTMLSpanElement, ChipProps>(
  ({className, variant, ...props}, ref) => (
    <span ref={ref} className={cn(chipVariants({variant}), className)} {...props} />
  ),
)
Chip.displayName = 'Chip'

export {chipVariants}
