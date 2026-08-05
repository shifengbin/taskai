import {forwardRef, type ComponentPropsWithoutRef, type ElementRef} from 'react'
import * as ToastPrimitive from '@radix-ui/react-toast'
import {cva, type VariantProps} from 'class-variance-authority'
import {X} from 'lucide-react'
import {cn, focusRing} from '../../lib/utils'

export const ToastProvider = ToastPrimitive.Provider

const toastVariants = cva(
  cn(
    'group pointer-events-auto relative flex w-full items-start gap-3 p-3 pr-9',
    'border-2 border-snap-outline rounded-snap bg-snap-surface text-snap-ink shadow-snap-lg',
    'data-[state=open]:animate-in data-[state=closed]:animate-out',
    'data-[state=open]:slide-in-from-bottom-3 data-[state=closed]:fade-out-80',
    'data-[swipe=move]:translate-x-[var(--radix-toast-swipe-move-x)]',
    'data-[swipe=cancel]:translate-x-0 data-[swipe=end]:translate-x-[var(--radix-toast-swipe-end-x)]',
  ),
  {
    variants: {
      variant: {
        default: '',
        success: '[&_[data-toast-icon]]:text-snap-cobalt',
        error: 'border-snap-error [&_[data-toast-icon]]:text-snap-error',
        warning: '[&_[data-toast-icon]]:text-snap-amber',
      },
    },
    defaultVariants: {variant: 'default'},
  },
)

const Toast = forwardRef<
  ElementRef<typeof ToastPrimitive.Root>,
  ComponentPropsWithoutRef<typeof ToastPrimitive.Root> & VariantProps<typeof toastVariants>
>(({className, variant, ...props}, ref) => (
  <ToastPrimitive.Root ref={ref} className={cn(toastVariants({variant}), className)} {...props} />
))
Toast.displayName = ToastPrimitive.Root.displayName

const ToastClose = forwardRef<
  ElementRef<typeof ToastPrimitive.Close>,
  ComponentPropsWithoutRef<typeof ToastPrimitive.Close>
>(({className, ...props}, ref) => (
  <ToastPrimitive.Close
    ref={ref}
    className={cn(
      'absolute right-1.5 top-1.5 inline-grid place-items-center h-6 w-6 rounded-snap-sm',
      'text-snap-muted transition-colors hover:bg-snap-surface-2 hover:text-snap-ink',
      focusRing,
      className,
    )}
    toast-close=""
    {...props}
  >
    <X className="h-3.5 w-3.5" strokeWidth={2.25} />
    <span className="sr-only">关闭</span>
  </ToastPrimitive.Close>
))
ToastClose.displayName = ToastPrimitive.Close.displayName

const ToastTitle = forwardRef<
  ElementRef<typeof ToastPrimitive.Title>,
  ComponentPropsWithoutRef<typeof ToastPrimitive.Title>
>(({className, ...props}, ref) => (
  <ToastPrimitive.Title
    ref={ref}
    className={cn('font-display text-sm font-extrabold text-snap-ink', className)}
    {...props}
  />
))
ToastTitle.displayName = ToastPrimitive.Title.displayName

const ToastDescription = forwardRef<
  ElementRef<typeof ToastPrimitive.Description>,
  ComponentPropsWithoutRef<typeof ToastPrimitive.Description>
>(({className, ...props}, ref) => (
  <ToastPrimitive.Description
    ref={ref}
    className={cn('text-xs text-snap-muted', className)}
    {...props}
  />
))
ToastDescription.displayName = ToastPrimitive.Description.displayName

const ToastViewport = forwardRef<
  ElementRef<typeof ToastPrimitive.Viewport>,
  ComponentPropsWithoutRef<typeof ToastPrimitive.Viewport>
>(({className, ...props}, ref) => (
  <ToastPrimitive.Viewport
    ref={ref}
    className={cn(
      'fixed bottom-4 right-4 z-[100] flex max-h-screen w-full flex-col gap-2 p-0 outline-none',
      'sm:max-w-sm',
      className,
    )}
    {...props}
  />
))
ToastViewport.displayName = ToastPrimitive.Viewport.displayName

export {
  Toast,
  ToastClose,
  ToastTitle,
  ToastDescription,
  ToastViewport,
  toastVariants,
}
