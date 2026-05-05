import type { VariantProps } from "class-variance-authority"
import { cva } from "class-variance-authority"

export { default as Button } from "./Button.vue"

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md border font-medium transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default:
          "bg-[#1f883d] hover:bg-[#1a7f37] active:bg-[#187232] text-white border-[rgba(31,35,40,0.15)] dark:bg-[#238636] dark:hover:bg-[#2ea043] dark:border-[rgba(240,246,252,0.1)]",
        secondary:
          "bg-[#f6f8fa] hover:bg-[#eff2f5] active:bg-[#e6eaef] text-[#1f2328] border-[#d1d9e0] dark:bg-[#21262d] dark:hover:bg-[#30363d] dark:text-[#e6edf3] dark:border-[#30363d]",
        outline:
          "bg-transparent text-[#0969da] hover:bg-[#ddf4ff] border-[#d1d9e0] dark:text-[#2f81f7] dark:hover:bg-[rgba(56,139,253,0.15)] dark:border-[#30363d]",
        destructive:
          "bg-transparent text-[#cf222e] hover:bg-[#cf222e] hover:text-white border-[#d1d9e0] dark:text-[#f85149] dark:hover:bg-[#da3633] dark:hover:text-white dark:border-[#30363d]",
        ghost:
          "bg-transparent border-transparent text-foreground hover:bg-muted",
        link:
          "bg-transparent border-transparent text-[#0969da] dark:text-[#2f81f7] underline-offset-4 hover:underline",
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
