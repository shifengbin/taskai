import {forwardRef, type ComponentPropsWithoutRef, type ElementRef} from 'react'
import * as AccordionPrimitive from '@radix-ui/react-accordion'
import {ChevronDown} from 'lucide-react'
import {cn, focusRing} from '../../lib/utils'

export const Accordion = AccordionPrimitive.Root

const AccordionItem = forwardRef<
  ElementRef<typeof AccordionPrimitive.Item>,
  ComponentPropsWithoutRef<typeof AccordionPrimitive.Item>
>(({className, ...props}, ref) => (
  <AccordionPrimitive.Item
    ref={ref}
    className={cn(
      'border border-snap-outline rounded-snap bg-snap-surface shadow-snap-sm overflow-hidden',
      className,
    )}
    {...props}
  />
))
AccordionItem.displayName = 'AccordionItem'

const AccordionTrigger = forwardRef<
  ElementRef<typeof AccordionPrimitive.Trigger>,
  ComponentPropsWithoutRef<typeof AccordionPrimitive.Trigger>
>(({className, children, ...props}, ref) => (
  <AccordionPrimitive.Header className="flex">
    <AccordionPrimitive.Trigger
      ref={ref}
      className={cn(
        'flex flex-1 items-center justify-between gap-2 px-3 py-2.5 text-left',
        'font-display font-bold text-sm text-snap-ink',
        'transition-colors hover:bg-snap-surface-2',
        '[&[data-state=open]>svg]:rotate-180',
        focusRing,
        className,
      )}
      {...props}
    >
      {children}
      <ChevronDown className="h-4 w-4 shrink-0 text-snap-muted transition-transform duration-200" strokeWidth={2.25} />
    </AccordionPrimitive.Trigger>
  </AccordionPrimitive.Header>
))
AccordionTrigger.displayName = 'AccordionTrigger'

const AccordionContent = forwardRef<
  ElementRef<typeof AccordionPrimitive.Content>,
  ComponentPropsWithoutRef<typeof AccordionPrimitive.Content>
>(({className, children, ...props}, ref) => (
  <AccordionPrimitive.Content
    ref={ref}
    className={cn(
      'overflow-hidden text-sm text-snap-ink',
      'data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down',
    )}
    {...props}
  >
    <div className={cn('px-3 pb-3 pt-1', className)}>{children}</div>
  </AccordionPrimitive.Content>
))
AccordionContent.displayName = 'AccordionContent'

export {AccordionItem, AccordionTrigger, AccordionContent}
