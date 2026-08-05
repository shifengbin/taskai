import {forwardRef} from 'react'
import * as SwitchPrimitive from '@radix-ui/react-switch'
import {cn} from '../../lib/utils'

const Switch = forwardRef<
  React.ElementRef<typeof SwitchPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitive.Root>
>(({className, ...props}, ref) => (
  <SwitchPrimitive.Root
    ref={ref}
    className={cn(
      'peer inline-flex h-6 w-11 shrink-0 cursor-pointer items-center',
      'border-2 border-snap-outline rounded-snap-sm bg-snap-surface-2 px-0.5',
      'transition-colors outline-none',
      'data-[state=checked]:bg-snap-cobalt',
      'focus-visible:ring-[3px] focus-visible:ring-snap-cobalt',
      'disabled:cursor-not-allowed disabled:opacity-50',
      className,
    )}
    {...props}
  >
    <SwitchPrimitive.Thumb
      className={cn(
        'pointer-events-none block h-4 w-4 border-2 border-snap-outline rounded-snap-sm bg-snap-surface',
        'shadow-snap-sm transition-transform',
        'data-[state=checked]:translate-x-5 data-[state=unchecked]:translate-x-0',
      )}
    />
  </SwitchPrimitive.Root>
))
Switch.displayName = SwitchPrimitive.Root.displayName

export {Switch}
