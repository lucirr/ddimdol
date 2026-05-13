import { cn } from '@/lib/utils'

type InputProps = React.InputHTMLAttributes<HTMLInputElement>

export function Input({ className, ...props }: InputProps) {
  return (
    <input
      className={cn(
        'w-full rounded-md border border-gray-300 px-3 py-2 text-sm',
        'focus:outline-none focus:ring-2 focus:ring-gray-400',
        'disabled:opacity-50',
        className,
      )}
      {...props}
    />
  )
}
