import {cloneElement, forwardRef, isValidElement, useId, type InputHTMLAttributes, type LabelHTMLAttributes, type ReactElement, type ReactNode, type TextareaHTMLAttributes} from 'react'
import {cn, focusRing} from '../../lib/utils'

const fieldBase = cn(
  'w-full border-2 border-snap-outline rounded-snap bg-snap-surface px-3 py-2',
  'text-sm text-snap-ink placeholder:text-snap-muted shadow-snap-sm',
  'transition-[box-shadow,border-color] focus-visible:border-snap-cobalt',
  focusRing,
  'disabled:cursor-not-allowed disabled:opacity-60',
)

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean
}

/** Snap Input — native input with 2px outline, cobalt focus ring, small hard shadow. */
export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({className, invalid, ...props}, ref) => (
    <input
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cn(fieldBase, invalid && 'border-snap-error focus-visible:border-snap-error', className)}
      {...props}
    />
  ),
)
Input.displayName = 'Input'

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  invalid?: boolean
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({className, invalid, ...props}, ref) => (
    <textarea
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cn(fieldBase, 'resize-y leading-relaxed', invalid && 'border-snap-error focus-visible:border-snap-error', className)}
      {...props}
    />
  ),
)
Textarea.displayName = 'Textarea'

export interface LabelProps extends LabelHTMLAttributes<HTMLLabelElement> {}

export const Label = forwardRef<HTMLLabelElement, LabelProps>(
  ({className, children, ...props}, ref) => (
    <label
      ref={ref}
      className={cn('font-display text-xs font-extrabold uppercase tracking-wide text-snap-ink', className)}
      {...props}
    >
      {children}
    </label>
  ),
)
Label.displayName = 'Label'

export interface FieldProps {
  label?: ReactNode
  htmlFor?: string
  hint?: ReactNode
  error?: ReactNode
  children: ReactNode
  className?: string
}

/** Label + control + hint/error wrapper, replacing MUI FormControlLabel/TextField scaffolding. */
export function Field({label, htmlFor, hint, error, children, className}: FieldProps) {
  const autoId = useId()
  const id = htmlFor ?? (label ? autoId : undefined)
  return (
    <div className={cn('grid gap-1.5', className)}>
      {label ? <Label htmlFor={id}>{label}</Label> : null}
      {label && id && isValidElement(children)
        ? cloneElement(children as ReactElement, {id} as {id?: string})
        : children}
      {error ? <p className="text-xs font-semibold text-snap-error">{error}</p> : null}
      {!error && hint ? <p className="text-xs text-snap-muted">{hint}</p> : null}
    </div>
  )
}
