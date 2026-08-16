The WSO2 action button — sentence-case, verb-first labels ("Get started", "Read the docs").

```jsx
<Button>Start for free</Button>
<Button variant="secondary">Talk to an expert</Button>
<Button variant="ghost" size="sm">Cancel</Button>
<Button iconRight={<i data-lucide="arrow-right" />}>Get started</Button>
```

Variants: `primary` (orange + soft glow — one per view), `secondary` (outlined, neutral), `ghost` (transparent, fills on hover), `inverse` (white, for dark heroes), `danger` (red). Sizes `sm`/`md`/`lg`. Supports `iconLeft`/`iconRight`, `loading`, `fullWidth`, `disabled`.
