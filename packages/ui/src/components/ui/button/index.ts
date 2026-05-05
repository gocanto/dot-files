import type { VariantProps } from "class-variance-authority"
import { cva } from "class-variance-authority"

export { default as Button } from "./Button.vue"

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md border font-medium transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default:
          "bg-success hover:bg-[var(--success-hover)] active:bg-[var(--success-active)] text-success-foreground border-[var(--btn-border-translucent)]",
        secondary:
          "bg-secondary hover:bg-[#eff2f5] active:bg-[#e6eaef] text-secondary-foreground border-border dark:hover:bg-[#262c36] dark:active:bg-[#2a313c]",
        outline:
          "bg-transparent text-primary hover:bg-primary hover:text-primary-foreground border-border",
        destructive:
          "bg-transparent text-destructive hover:bg-destructive hover:text-white border-border dark:hover:bg-[#b62324]",
        ghost:
          "bg-transparent border-transparent text-foreground hover:bg-muted",
        link:
          "bg-transparent border-transparent text-primary underline-offset-4 hover:underline",
      },
      size: {
        "default": "h-8 text-sm px-3 has-[>svg]:px-2",
        "sm": "h-7 text-xs gap-1.5 px-2 has-[>svg]:px-2",
        "lg": "h-10 text-base px-4 has-[>svg]:px-3",
        "icon": "size-8",
        "icon-sm": "size-7",
        "icon-lg": "size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
)
export type ButtonVariants = VariantProps<typeof buttonVariants>
