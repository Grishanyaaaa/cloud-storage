import { forwardRef, type TextareaHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(
        "flex min-h-[80px] w-full rounded-md border border-border-1 bg-bg-4 px-3 py-2 text-sm",
        "text-fg-1 placeholder:text-fg-3",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-1",
        "focus-visible:ring-offset-2 focus-visible:ring-offset-bg-0",
        "disabled:cursor-not-allowed disabled:opacity-50",
        "transition-colors resize-none",
        className,
      )}
      {...props}
    />
  ),
);
Textarea.displayName = "Textarea";
