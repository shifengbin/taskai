import {forwardRef} from 'react'
import * as CheckboxPrimitive from '@radix-ui/react-checkbox'
import {Check, Minus} from 'lucide-react'
import {cn} from '../../lib/utils'

const Checkbox = forwardRef<
  React.ElementRef<typeof CheckboxPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>
>(({className, ...props}, ref) => (
  <CheckboxPrimitive.Root
    ref={ref}
    className={cn(
      'peer h-5 w-5 shrink-0 border border-snap-outline rounded-snap-sm bg-snap-surface',
      'outline-none transition-colors',
      'data-[state=checked]:bg-snap-cobalt data-[state=checked]:text-white',
      'data-[state=indeterminate]:bg-snap-cobalt data-[state=indeterminate]:text-white',
      'focus-visible:ring-[3px] focus-visible:ring-snap-cobalt',
      'disabled:cursor-not-allowed disabled:opacity-50',
      className,
    )}
    {...props}
  >
    <CheckboxPrimitive.Indicator className="flex items-center justify-center text-current">
      {props.checked === 'indeterminate' ? (
        <Minus className="h-3.5 w-3.5" strokeWidth={3} />
      ) : (
        <Check className="h-3.5 w-3.5" strokeWidth={3} />
      )}
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
))
Checkbox.displayName = CheckboxPrimitive.Root.displayName

export {Checkbox}
