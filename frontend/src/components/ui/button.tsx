import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { forwardRef, type ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

const buttonVariants = cva(
  cn(
    "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium",
    "transition-colors disabled:pointer-events-none disabled:opacity-50",
    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-1",
    "focus-visible:ring-offset-2 focus-visible:ring-offset-bg-0",
  ),
  {
    variants: {
      intent: {
        primary: "bg-accent-1 text-white hover:bg-accent-1h active:bg-accent-1p",
        secondary: "bg-bg-3 text-fg-1 border border-border-1 hover:bg-bg-4",
        ghost: "text-fg-1 hover:bg-bg-2",
        danger: "bg-danger text-white hover:opacity-90",
        link: "text-accent-1 underline-offset-4 hover:underline",
      },
      size: {
        sm: "h-7 px-2 text-[13px]",
        md: "h-9 px-3 text-sm",
        lg: "h-11 px-4 text-base",
        icon: "h-9 w-9 p-0",
      },
    },
    defaultVariants: { intent: "primary", size: "md" },
  },
);

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  };

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, intent, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp className={cn(buttonVariants({ intent, size }), className)} ref={ref} {...props} />
    );
  },
);
Button.displayName = "Button";

export { buttonVariants };
