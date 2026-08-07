import {forwardRef, type HTMLAttributes} from 'react'
import {cva, type VariantProps} from 'class-variance-authority'
import {cn} from '../../lib/utils'

const alertVariants = cva(
  'relative flex w-full gap-3 p-3 border border-snap-outline rounded-snap bg-snap-surface shadow-snap-sm text-sm text-snap-ink',
  {
    variants: {
      severity: {
        info: '[&_[data-alert-icon]]:text-snap-cobalt',
        success: '[&_[data-alert-icon]]:text-snap-cobalt',
        warning: '[&_[data-alert-icon]]:text-snap-amber',
        error: '[&_[data-alert-icon]]:text-snap-error',
      },
    },
    defaultVariants: {severity: 'info'},
  },
)

export interface AlertProps
  extends HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof alertVariants> {
  icon?: React.ReactNode
}

/** Snap Alert — hard-edged feedback box; severity tints the optional icon. */
export const Alert = forwardRef<HTMLDivElement, AlertProps>(
  ({className, severity, icon, children, ...props}, ref) => (
    <div ref={ref} role="alert" className={cn(alertVariants({severity}), className)} {...props}>
      {icon ? <span data-alert-icon className="shrink-0">{icon}</span> : null}
      <div className="grid gap-1">{children}</div>
    </div>
  ),
)
Alert.displayName = 'Alert'

export interface AlertTitleProps extends HTMLAttributes<HTMLHeadingElement> {}

export const AlertTitle = forwardRef<HTMLHeadingElement, AlertTitleProps>(
  ({className, ...props}, ref) => (
    <h5 ref={ref} className={cn('font-display font-extrabold leading-none', className)} {...props} />
  ),
)
AlertTitle.displayName = 'AlertTitle'

export {alertVariants}
