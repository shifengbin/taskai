import {forwardRef, type ComponentPropsWithoutRef, type ElementRef} from 'react'
import * as TabsPrimitive from '@radix-ui/react-tabs'
import {cn, focusRing} from '../../lib/utils'

export const Tabs = TabsPrimitive.Root

const TabsList = forwardRef<
  ElementRef<typeof TabsPrimitive.List>,
  ComponentPropsWithoutRef<typeof TabsPrimitive.List>
>(({className, ...props}, ref) => (
  <TabsPrimitive.List
    ref={ref}
    className={cn('flex items-stretch w-full border-b border-snap-outline bg-snap-surface', className)}
    {...props}
  />
))
TabsList.displayName = TabsPrimitive.List.displayName

const TabsTrigger = forwardRef<
  ElementRef<typeof TabsPrimitive.Trigger>,
  ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>
>(({className, ...props}, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    className={cn(
      'inline-flex flex-1 items-center justify-center gap-1.5 whitespace-nowrap',
      'h-10 px-4 font-display font-bold text-xs uppercase tracking-wide text-snap-muted',
      'border-r border-snap-outline transition-colors last:border-r-0',
      'data-[state=active]:bg-snap-surface-2 data-[state=active]:text-snap-ink data-[state=active]:shadow-[inset_0_-2px_0_0_var(--snap-cobalt)]',
      focusRing,
      'disabled:pointer-events-none disabled:opacity-50',
      className,
    )}
    {...props}
  />
))
TabsTrigger.displayName = TabsPrimitive.Trigger.displayName

const TabsContent = forwardRef<
  ElementRef<typeof TabsPrimitive.Content>,
  ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({className, ...props}, ref) => (
  <TabsPrimitive.Content
    ref={ref}
    className={cn(focusRing, 'flex-1 min-h-0 outline-none', className)}
    {...props}
  />
))
TabsContent.displayName = TabsPrimitive.Content.displayName

export {TabsList, TabsTrigger, TabsContent}
